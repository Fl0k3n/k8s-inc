package controller

import (
	"context"
	"fmt"

	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (m *MicroSdn) GetTopology(context.Context, *emptypb.Empty) (*pb.TopologyResponse, error) {
	fmt.Println("Handling getTopology")
	return TopologyModelToDao(m.topo), nil
}

