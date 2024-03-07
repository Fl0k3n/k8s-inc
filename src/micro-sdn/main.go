package main

import (
	"fmt"
	"net"

	"github.com/Fl0k3n/k8s-inc/micro-sdn/controller"
	"github.com/Fl0k3n/k8s-inc/micro-sdn/generated"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	"google.golang.org/grpc"
)

func runServer(frontend *controller.MicroSdn, grpcAddr string) error {
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
	microSdn := controller.NewMicroSdn(topo)

	err := runServer(microSdn, "127.0.0.1:9001")
	if err != nil {
		fmt.Printf("Failed to run gRpc server: %e\n", err)
	}
}
