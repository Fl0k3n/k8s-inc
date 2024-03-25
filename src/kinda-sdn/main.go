package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
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

func updateNames(topo *model.Topology) {
	// kubectl get nodes -l sname=w1 --no-headers 
	nodeNamesRemap := map[model.DeviceName]model.DeviceName{}
	for i := 0; i < len(topo.Devices); i++ {
		dev := topo.Devices[i]
		if dev.GetType() == model.NODE {
			n := dev.(*model.Host)
			cmd := exec.Command("kubectl", "get", "node", "-l", "sname=" + string(n.Name), "--no-headers")
			out, err := cmd.Output()
			if err != nil {
				panic(err)
			}
			oldName := n.Name
			n.Name = model.DeviceName(strings.Split(string(out), " ")[0])
			nodeNamesRemap[oldName] = n.Name
			topo.Devices[i] = n
		}
	}
	for _, dev := range topo.Devices {
		for _, link := range dev.GetLinks() {
			if newName, ok := nodeNamesRemap[link.To]; ok {
				link.To = newName
			}
		}
	}
}

func main() {
	topo := generated.V4_gRpc_topo()
	// updateNames(topo)
	// p4Config := generated.V3_grpc_p4_conf_raw(false)
	telemetryService := telemetry.NewTelemetryService()
	// kindaSdn := controller.NewKindaSdn(topo, p4Config, telemetryService)
	kindaSdn := controller.NewKindaSdn(topo, map[string][]connector.RawTableEntry{}, telemetryService)
	fmt.Println("Initializing topology")
	if err := kindaSdn.InitTopology(true); err != nil {
		fmt.Println("Failed to init topology")
		fmt.Println(err)
		return
	}
	fmt.Println("Running gRpc server")

	err := runServer(kindaSdn, "127.0.0.1:9001")
	if err != nil {
		fmt.Printf("Failed to run gRpc server: %e\n", err)
		fmt.Println(err)
	}
}
