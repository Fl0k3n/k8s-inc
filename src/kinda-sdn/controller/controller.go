package controller

import (
	"context"
	"fmt"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/device"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/telemetry"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
)

type KindaSdn struct {
	pb.UnimplementedSdnFrontendServer
	pbt.UnimplementedTelemetryServiceServer
	topo *model.Topology
	initialP4Config map[model.DeviceName][]connector.RawTableEntry
	telemetryService *telemetry.TelemetryService
	bmv2Managers map[model.DeviceName]*device.Bmv2Manager
}

func NewKindaSdn(
	topo *model.Topology,
	initialP4Config map[model.DeviceName][]connector.RawTableEntry,
	telemetryService *telemetry.TelemetryService,
) *KindaSdn {
	return &KindaSdn{
		topo: topo,
		initialP4Config: initialP4Config,
		telemetryService: telemetryService,
		bmv2Managers: make(map[string]*device.Bmv2Manager),
	}
}

func (k *KindaSdn) Close() {
	for _, bmv2 := range k.bmv2Managers {
		bmv2.Close()
	}
}

func (k *KindaSdn) bootstrapSwitch(incSwitch *model.IncSwitch) error {
	ctx := context.Background()
	if incSwitch.Arch != model.BMv2 {
		panic("must be bmv2")	
	}
	bmv2 := device.NewBmv2Manager(incSwitch.GrpcUrl)
	if err := bmv2.Open(ctx); err != nil {
		return err
	}
	k.bmv2Managers[incSwitch.Name] = bmv2

	binPath, p4infoPath := generated.V3_telemetry_artifact_paths()
	if err := bmv2.InstallProgram(binPath, p4infoPath); err != nil{
		return err
	}
	incSwitch.InstalledProgram = "telemetry"

	entries := k.initialP4Config[incSwitch.Name]
	if err := bmv2.WriteInitialEntries(ctx, entries); err != nil {
		return err
	}
	return nil
}

func (k *KindaSdn) InitTopology() error {
	for _, dev := range k.topo.Devices {
		if dev.GetType() == model.INC_SWITCH {
			if err := k.bootstrapSwitch(dev.(*model.IncSwitch)); err != nil {
				fmt.Printf("Failed to write entries for switch %s\n", dev.GetName())
				return err
			}
		}
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
