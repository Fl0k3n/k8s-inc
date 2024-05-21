package controller

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/device"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/programs"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/telemetry"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
)

type KindaSdn struct {
	pb.UnimplementedSdnFrontendServer
	pbt.UnimplementedTelemetryServiceServer
	programRegistry programs.P4ProgramRegistry
	topo *model.Topology
	initialP4Config map[model.DeviceName][]connector.RawTableEntry
	telemetryService *telemetry.TelemetryService
	bmv2Managers map[model.DeviceName]*device.Bmv2Manager

	// TODO this is just for evaluation purposes, a lot of work is required here to 
	// deal with concurrency properly
	networkUpdatesChan chan *pb.NetworkChange
	numUpdateListeners *atomic.Int32
}

func NewKindaSdn(
	topo *model.Topology,
	programRegistry programs.P4ProgramRegistry,
	initialP4Config map[model.DeviceName][]connector.RawTableEntry,
	telemetryService *telemetry.TelemetryService,
) *KindaSdn {
	numUpdateListeners := &atomic.Int32{}
	numUpdateListeners.Store(0)

	return &KindaSdn{
		topo: topo,
		programRegistry: programRegistry,
		initialP4Config: initialP4Config,
		telemetryService: telemetryService,
		bmv2Managers: make(map[string]*device.Bmv2Manager),
		networkUpdatesChan: make(chan *pb.NetworkChange),
		numUpdateListeners: numUpdateListeners,
	}
}

func (k *KindaSdn) Close() {
	for _, bmv2 := range k.bmv2Managers {
		bmv2.Close()
	}
}

func (k *KindaSdn) bootstrapSwitch(incSwitch *model.IncSwitch) error {
	programDetails, ok := k.programRegistry.Lookup(incSwitch.InstalledProgram)
	if !ok {
		return fmt.Errorf("program %s not registered but required on device %s", incSwitch.InstalledProgram, incSwitch.Name)
	}
	ctx := context.Background()
	if incSwitch.Arch != model.BMv2 {
		panic("must be bmv2")
	}
	artifacts, ok := programDetails.GetArtifactsFor(incSwitch.Arch) 
	if !ok {
		panic("couldn't find artifacts")
	}

	bmv2 := device.NewBmv2Manager(incSwitch.GrpcUrl)
	if err := bmv2.Open(ctx); err != nil {
		return err
	}
	k.bmv2Managers[incSwitch.Name] = bmv2

	if err := bmv2.InstallProgram(artifacts.P4PipelinePath, artifacts.P4InfoPath); err != nil{
		return err
	}
	incSwitch.InstalledProgram = programDetails.Name
	return nil
}

func (k *KindaSdn) writeInitialEntriesToBmv2Switches(entries map[model.DeviceName][]connector.RawTableEntry) error {
	ctx := context.Background()
	for devName, ents := range entries {
		if err := k.bmv2Managers[devName].WriteInitialEntries(ctx, ents); err != nil {
			return err
		}
	}
	return nil
}

func (k *KindaSdn) notifyNetworkChanged (change *pb.NetworkChange) {
	go func ()  {
		for {
			select {
			case k.networkUpdatesChan <- change:
				return
			default:
				if k.numUpdateListeners.Load() == 0 {
					// ignore this update, queue is full
					return
				} else {
					// TODO use condVar 
					time.Sleep(100*time.Millisecond)
				}
			}
		}
	}()
}

func (k *KindaSdn) AddDevices(devices []model.Device) error {
	k.topo.Devices = append(k.topo.Devices, devices...)
	G := model.TopologyToGraph(k.topo)
	for _, dev := range devices {
		for _, link := range dev.GetLinks() {
			neigh, ok := G[link.To]
			if !ok {
				return fmt.Errorf("device %s links to device %s which doesn't exist", dev.GetName(), link.To)
			}
			neigh.AddLink(model.NewLink(dev.GetName(), "00:00:00:00:00:00", "00.00.00.00", 0))
		}
	}
	k.notifyNetworkChanged(&pb.NetworkChange{
		Change: &pb.NetworkChange_TopologyChange{},
	})
	return nil
}

func (k *KindaSdn) RemoveDevices(deviceNames []model.DeviceName) error {
	deletedDevices := []model.Device{}
	k.topo.Devices = slices.DeleteFunc(k.topo.Devices, func(d model.Device) bool {
		if slices.Contains(deviceNames, d.GetName()) {
			deletedDevices = append(deletedDevices, d)
			return true
		}
		return false
	})
	if len(deletedDevices) < len(deviceNames) {
		return fmt.Errorf("some device wasn't deleted")
	}
	G := model.TopologyToGraph(k.topo)
	for _, dev := range deletedDevices {
		for _, link	:= range dev.GetLinks() {
			G[link.To].RemoveLink(dev.GetName())
		}
	}
	k.notifyNetworkChanged(&pb.NetworkChange{
		Change: &pb.NetworkChange_TopologyChange{},
	})
	return nil
}

func (k *KindaSdn) ChangeProgram(deviceName model.DeviceName, programName string) error {
	for _, dev := range k.topo.Devices {
		if dev.GetName() == deviceName {
			sw, ok := dev.(*model.IncSwitch)
			if !ok {
				return fmt.Errorf("device %s is not an inc switch, can't update program", deviceName)
			}
			sw.InstalledProgram = programName
			k.notifyNetworkChanged(&pb.NetworkChange{
				Change: &pb.NetworkChange_ProgramChange{},
			})
			return nil
		}
	}
	return fmt.Errorf("device %s doesn't exist", deviceName)
}

func (k *KindaSdn) InitTopology(setupL3Forwarding bool) error {
	entries := k.initialP4Config
	for _, dev := range k.topo.Devices {
		if dev.GetType() == model.INC_SWITCH {
			if err := k.bootstrapSwitch(dev.(*model.IncSwitch)); err != nil {
				fmt.Printf("Failed to write entries for switch %s\n", dev.GetName())
				return err
			}
		}
	}
	if setupL3Forwarding {
		l3Entries := k.buildBasicForwardingEntries()
		for devName, ents := range l3Entries {
			if presentEntries, arePresent := entries[devName]; arePresent {
				entries[devName] = append(presentEntries, ents...)
			} else {
				entries[devName] = ents
			}
		}
	}
	if err := k.writeInitialEntriesToBmv2Switches(entries); err != nil {
		return err
	}

	ctx := context.Background()
	if err := k.telemetryService.InitDevices(ctx, k.topo, func(dn model.DeviceName) device.IncSwitch {
		return k.bmv2Managers[dn]
	}); err != nil {
		fmt.Printf("Telemetry service init failed %e\n", err)
		return err
	}
	return nil
}

