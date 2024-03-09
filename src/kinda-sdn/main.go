package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	"google.golang.org/grpc"
)

func runServer(frontend *controller.KindaSdn, grpcAddr string) error {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	pb.RegisterSdnFrontendServer(server, frontend)
	return server.Serve(lis)
}

func updateNames(topo *model.Topology) {
	// kubectl get nodes -l sname=w1 --no-headers 
	for i := 0; i < len(topo.Devices); i++ {
		dev := topo.Devices[i]
		if dev.GetType() == model.NODE {
			n := dev.(*model.Host)
			cmd := exec.Command("kubectl", "get", "node", "-l", "sname=" + string(n.Name), "--no-headers")
			out, err := cmd.Output()
			if err != nil {
				panic(err)
			}
			n.Name = model.DeviceName(strings.Split(string(out), " ")[0])
			topo.Devices[i] = n
		}
	}
}

func main() {
	topo := generated.V3_grpc_topo()
	updateNames(topo)
	p4Config := generated.V3_grpc_p4_conf_raw()
	kindaSdn := controller.NewKindaSdn(topo, p4Config)
	if err := kindaSdn.InitTopology(); err != nil {
		fmt.Println("Failed to init topology")
		fmt.Println(err)
		return
	}

	err := runServer(kindaSdn, "127.0.0.1:9001")
	if err != nil {
		fmt.Printf("Failed to run gRpc server: %e\n", err)
		fmt.Println(err)
	}
}
