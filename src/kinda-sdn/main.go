package main

import (
	"fmt"
	"net"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
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

func main() {
	// topo := generated.V3_grpc_topo()
	topo, programDefinitions := generated.Measure_gRPC_topo()
	programRegistry := programs.NewRegistry()
	telemetryService := telemetry.NewService(programRegistry)
	for _, program := range programDefinitions {
		programRegistry.Register(*program, telemetryService)
	}
	kindaSdn := controller.NewKindaSdn(topo, programRegistry, map[string][]connector.RawTableEntry{}, telemetryService)
	fmt.Println("Initializing topology")
	if err := kindaSdn.InitTopology(true); err != nil {
		fmt.Println("Failed to init topology")
		fmt.Println(err)
		return
	}
	fmt.Println("Running gRpc server")
	// sprt := int32(7676)
	// dprt := int32(7878)
	// _, er := kindaSdn.ConfigureTelemetry(context.Background(), &pbt.ConfigureTelemetryRequest{
	// 	IntentId: "i1",
	// 	CollectionId: "asd",
	// 	CollectorNodeName: "w5",
	// 	CollectorPort: 6000,
	// 	Sources: &pbt.ConfigureTelemetryRequest_RawSources{
	// 		RawSources: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
	// 			{DeviceName: "w1", Port: &sprt},
	// 		},
	// 	}},
	// 	Targets: &pbt.ConfigureTelemetryRequest_RawTargets{
	// 		RawTargets: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
	// 			{DeviceName: "w3", Port: &dprt},
	// 		}},
	// 	},
	// })
	// if er != nil {
	// 	panic(er)
	// }
	// _, er = kindaSdn.ConfigureTelemetry(context.Background(), &pbt.ConfigureTelemetryRequest{
	// 	IntentId: "i2",
	// 	CollectionId: "asd",
	// 	CollectorNodeName: "w5",
	// 	CollectorPort: 6000,
	// 	Sources: &pbt.ConfigureTelemetryRequest_RawSources{
	// 		RawSources: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
	// 			{DeviceName: "w1", Port: &sprt},
	// 		},
	// 	}},
	// 	Targets: &pbt.ConfigureTelemetryRequest_RawTargets{
	// 		RawTargets: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
	// 			{DeviceName: "w3", Port: &dprt},
	// 		}},
	// 	},
	// })
	// if er != nil {
	// 	panic(er)
	// }
	// resp, er := kindaSdn.DisableTelemetry(context.Background(), &pbt.DisableTelemetryRequest{
	// 	IntentId: "i1",
	// })
	// if er != nil || resp.ShouldRetryLater {
	// 	panic(er)
	// }
	// time.Sleep(time.Second * 2)
	// resp, er = kindaSdn.DisableTelemetry(context.Background(), &pbt.DisableTelemetryRequest{
	// 	IntentId: "i2",
	// })
	// if er != nil || resp.ShouldRetryLater {
	// 	panic(er)
	// }
	// sprt += 1
	// dprt += 1
	// _, er = kindaSdn.ConfigureTelemetry(context.Background(), &pbt.ConfigureTelemetryRequest{
	// 	CollectionId: "asd2",
	// 	CollectorNodeName: "w4",
	// 	CollectorPort: 6001,
	// 	Sources: &pbt.ConfigureTelemetryRequest_RawSources{
	// 		RawSources: &pbt.RawTelemetryEntities{Entities: []*pbt.RawTelemetryEntity{
	// 			{DeviceName: "w1", Port: &sprt},
	// 		},
	// 	}},
	// 	Targets: &pbt.ConfigureTelemetryRequest_RawTargets{
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
