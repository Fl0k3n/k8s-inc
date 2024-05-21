package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/programs"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/telemetry"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var addr = flag.String("address", "127.0.0.1", "IP address on which services should be run")
var grpcPort = flag.Int("grpc-port", 9001, "Port on which gRPC service should be run")
var httpPort = flag.Int("http-port", 9002, "Port on which HTTP service should be run")

var evalReconciliation = flag.Bool("evaluate-reconciliation", false, "Operate on fake cluster without connecting to any device")
var fatTreeTopoK = flag.Int("fat-tree-k", -1, "K parameter of a fat tree topology that should be created for evaluation, should be used only in evaluate-reconciliation mode")
var fatTreeIncSwitchFraction = flag.Float64("inc-switch-fraction", 0.4, "[0, 1] number describing the fraction of INC switches that should be generated in evaluation topology")

func runGrpcServer(frontend *controller.KindaSdn, grpcAddr string) error {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	kaep := keepalive.EnforcementPolicy{
		MinTime:             5*time.Second,
		PermitWithoutStream: true,
	}
	kasp := keepalive.ServerParameters{
		Time:                  15 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               5 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	}
	server := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
	pb.RegisterSdnFrontendServer(server, frontend)
	pbt.RegisterTelemetryServiceServer(server, frontend)
	return server.Serve(lis)
}

func runHttpNetworkChangeServer(controller *controller.KindaSdn, addr string) {
	http.HandleFunc("/add-devices", controller.AddDevicesHandler)
	http.HandleFunc("/change-program", controller.ChangeProgramHandler)
	http.HandleFunc("/delete-device", controller.DeleteDeviceHandler)
	http.HandleFunc("/health", controller.HealthCheckHandler)
	fmt.Printf("Listenning for network change requests on HTTP: %s\n", addr)
	err := http.ListenAndServe(addr, nil)
	fmt.Printf("HTTP listener error: %v\n", err)
}

func runInReconciliationEvaluationMode() {
	topo, programDefinitions := generated.FatTreeFakeCluster(*fatTreeTopoK, float32(*fatTreeIncSwitchFraction))
	programRegistry := programs.NewRegistry()
	telemetryService := telemetry.NewService(programRegistry)
	for _, program := range programDefinitions {
		programRegistry.Register(*program, telemetryService)
	}
	kindaSdn := controller.NewKindaSdn(topo, programRegistry, map[string][]connector.RawTableEntry{}, telemetryService)
	go runHttpNetworkChangeServer(kindaSdn, fmt.Sprintf("%s:%d", *addr, *httpPort))
	err := runGrpcServer(kindaSdn, fmt.Sprintf("%s:%d", *addr, *grpcPort))
	if err != nil {
		fmt.Printf("Failed to run gRpc server: %e\n", err)
		fmt.Println(err)
	}	
}

func run() {
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
	go runHttpNetworkChangeServer(kindaSdn, fmt.Sprintf("%s:%d", *addr, *httpPort))
	fmt.Println("Running gRpc server")
	err := runGrpcServer(kindaSdn, fmt.Sprintf("%s:%d", *addr, *grpcPort))
	if err != nil {
		fmt.Printf("Failed to run gRpc server: %e\n", err)
		fmt.Println(err)
	}
}

func main() {
	flag.Parse()
	if *evalReconciliation { 
		k := *fatTreeTopoK
		if k <= 0 || k % 2 == 1 {
			panic("k for fat tree toplogy must be even and positive")
		}
		f := *fatTreeIncSwitchFraction
		if f < 0 || f > 1 {
			panic("fraction of inc switches for fat tree topology must be between 0 and 1")
		}
		runInReconciliationEvaluationMode()
	} else {
		run()
	}
}
