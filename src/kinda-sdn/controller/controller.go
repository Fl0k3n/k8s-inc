package controller

import (
	"context"
	"fmt"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
)

type KindaSdn struct {
	pb.UnimplementedSdnFrontendServer
	topo *model.Topology
	initialP4Config map[model.DeviceName][]connector.RawTableEntry
}

func NewKindaSdn(topo *model.Topology, initialP4Config map[model.DeviceName][]connector.RawTableEntry) *KindaSdn {
	return &KindaSdn{
		topo: topo,
		initialP4Config: initialP4Config,
	}
}

func (k *KindaSdn) bootstrapSwitch(incSwitch *model.IncSwitch) error {
	ctx := context.Background()
	if incSwitch.Arch != model.BMv2 {
		panic("must be bmv2")	
	}
	bmv2 := NewBmv2Manager(incSwitch.GrpcUrl)
	if err := bmv2.Open(ctx); err != nil {
		return err
	}
	defer bmv2.Close()
	binPath, p4infoPath := generated.V3_telemetry_artifact_paths()
	if err := bmv2.InstallProgram(binPath, p4infoPath); err != nil{
		return err
	}
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
	return nil
}


