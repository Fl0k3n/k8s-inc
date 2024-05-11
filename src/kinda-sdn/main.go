package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/programs"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/telemetry"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	"google.golang.org/grpc"
)

func runGrpcServer(frontend *controller.KindaSdn, grpcAddr string) error {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	pb.RegisterSdnFrontendServer(server, frontend)
	pbt.RegisterTelemetryServiceServer(server, frontend)
	return server.Serve(lis)
}

func runHttpNetworkChangeServer(controller *controller.KindaSdn, addr string) {
	http.HandleFunc("/add-devices", controller.AddDevicesHandler)
	http.HandleFunc("/change-program", controller.ChangeProgramHandler)
	fmt.Printf("Listenning for network change requests on HTTP: %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	fmt.Printf("HTTP listener error: %v\n", err)
}

func main() {
	topo, programDefinitions := generated.Measure_gRPC_topo()
	programRegistry := programs.NewRegistry()
	telemetryService := telemetry.NewService(programRegistry)
	for _, program := range programDefinitions {
		programRegistry.Register(*program, telemetryService)
	}
	kindaSdn := controller.NewKindaSdn(topo, programRegistry, map[string][]connector.RawTableEntry{}, telemetryService)
	fmt.Println("Initializing topology")
	// if err := kindaSdn.InitTopology(true); err != nil {
	// 	fmt.Println("Failed to init topology")
	// 	fmt.Println(err)
	// 	return
	// }
	go runHttpNetworkChangeServer(kindaSdn, "127.0.0.1:9002")
	fmt.Println("Running gRpc server")
	err := runGrpcServer(kindaSdn, "127.0.0.1:9001")
	if err != nil {
		fmt.Printf("Failed to run gRpc server: %e\n", err)
		fmt.Println(err)
	}
}
