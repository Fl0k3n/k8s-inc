package controller

import (
	"context"
	"fmt"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (m *KindaSdn) GetTopology(context.Context, *emptypb.Empty) (*pb.TopologyResponse, error) {
	fmt.Println("Handling getTopology")
	return TopologyModelToDao(m.topo), nil
}

func (m *KindaSdn) GetSwitchDetails(ctx context.Context, names *pb.SwitchNames) (*pb.SwitchDetailsResponse, error) {
	fmt.Println("Handling getSwitchDetails")
	res := map[string]*pb.SwitchDetails{}
	for _, name := range names.Names {
		var device model.Device = nil
		for _, dev := range m.topo.Devices {
			if dev.GetName() == model.DeviceName(name) {
				device = dev
				break
			}
		}
		if device == nil {
			return nil, status.Errorf(codes.NotFound, "device %s not found", name)
		}
		if incSwitch, ok := device.(*model.IncSwitch); ok {
			res[name] = IncSwitchToDetailsDao(incSwitch)
		} else {
			return nil, status.Errorf(codes.InvalidArgument, "device %s is not IncSwitch", name)
		}
	}
	return &pb.SwitchDetailsResponse{
		Details: res,
	}, nil
}
