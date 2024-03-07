package main

import (
	"fmt"
	"net"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/generated"
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

func main() {
	topo := generated.V3_grpc_topo()
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
