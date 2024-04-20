package main

import (
	"context"
	"fmt"
	"net"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/programs"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/telemetry"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	"google.golang.org/grpc"
)

func runServer(frontend *controller.KindaSdn, grpcAddr string) error {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	pb.RegisterSdnFrontendServer(server, frontend)
	pbt.RegisterTelemetryServiceServer(server, frontend)
	return server.Serve(lis)
}

func getDefaultTelemetryProgramDetails() *model.P4ProgramDetails {
	defaultProgramName := "telemetry-v5"
	interfaces := []string{telemetry.PROGRAM_INTERFACE}	

	binPath, p4InfoPath := generated.V3_telemetry_artifact_paths()
	return model.NewProgramDetails(
		defaultProgramName,
		interfaces,
		[]model.P4Artifacts{
			{
				Arch: model.BMv2,
				P4InfoPath: p4InfoPath,
				P4PipelinePath: binPath,
			},
		},
	)
}

func main() {
	// topo := generated.V3_grpc_topo()
	topo := generated.Measure_gRPC_topo()
	defaultProgram := getDefaultTelemetryProgramDetails()
	programRegistry := programs.NewRegistry()
	telemetryService := telemetry.NewService(programRegistry)
	programRegistry.Register(*defaultProgram, telemetryService)

	kindaSdn := controller.NewKindaSdn(topo, programRegistry, map[string][]connector.RawTableEntry{}, telemetryService)
	fmt.Println("Initializing topology")
	if err := kindaSdn.InitTopology(true, defaultProgram.Name); err != nil {
		fmt.Println("Failed to init topology")
		fmt.Println(err)
		return
	}
	fmt.Println("Running gRpc server")
	sprt := int32(7676)
	dprt := int32(7878)
	_, er := kindaSdn.EnableTelemetry(context.Background(), &pbt.EnableTelemetryRequest{
		IntentId: "i1",
		CollectionId: "asd",
		CollectorNodeName: "w5",
		CollectorPort: 6000,
		Sources: &pbt.EnableTelemetryRequest_RawSources{
			RawSources: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
				{DeviceName: "w1", Port: &sprt},
			},
		}},
		Targets: &pbt.EnableTelemetryRequest_RawTargets{
			RawTargets: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
				{DeviceName: "w3", Port: &dprt},
			}},
		},
	})
	if er != nil {
		panic(er)
	}
	// sprt += 1
	// dprt += 1
	// _, er = kindaSdn.EnableTelemetry(context.Background(), &pbt.EnableTelemetryRequest{
	// 	CollectionId: "asd2",
	// 	CollectorNodeName: "w4",
	// 	CollectorPort: 6001,
	// 	Sources: &pbt.EnableTelemetryRequest_RawSources{
	// 		RawSources: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
	// 			{DeviceName: "w1", Port: &sprt},
	// 		},
	// 	}},
	// 	Targets: &pbt.EnableTelemetryRequest_RawTargets{
	// 		RawTargets: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
	// 			{DeviceName: "w3", Port: &dprt},
	// 		}},
	// 	},
	// })
	// if er != nil {
	// 	panic(er)
	// }

	err := runServer(kindaSdn, "127.0.0.1:9001")
	if err != nil {
		fmt.Printf("Failed to run gRpc server: %e\n", err)
		fmt.Println(err)
	}
}
