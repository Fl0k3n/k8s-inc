package controller

import (
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
)

func LinkToProto(link *model.Link) *pb.Link {
	return &pb.Link{
		PeerName: string(link.To),
	}
}

func convertDeviceType(device model.Device) pb.DeviceType {
	var dt pb.DeviceType
	switch device.GetType() {
	case model.NET:
		dt = pb.DeviceType_NET
	case model.EXTERNAL:
		dt = pb.DeviceType_EXTERNAL
	case model.INC_SWITCH:
		if device.(*model.IncSwitch).AllowClientProgrammability {
			dt = pb.DeviceType_INC_SWITCH
		} else {
			dt = pb.DeviceType_NET
		}
	case model.NODE:
		dt = pb.DeviceType_HOST
	}
	return dt
}

func DeviceToProto(device model.Device) *pb.Device {
	links := make([]*pb.Link, len(device.GetLinks()))
	for i, link := range device.GetLinks() {
		links[i] = LinkToProto(link)
	}
	return &pb.Device{
		Index: int32(device.GetIndex()),
		Name: string(device.GetName()),
		DeviceType:  convertDeviceType(device),
		Links: links,
	}
}

func TopologyModelToProto(topo *model.Topology) *pb.TopologyResponse {
	devices := make([]*pb.Device, len(topo.Devices))
	for i, dev := range topo.Devices {
		devices[i] = DeviceToProto(dev)
	}
	return &pb.TopologyResponse{
		Graph: devices,
	}
}

func IncSwitchToDetailsProto(s *model.IncSwitch) *pb.SwitchDetails {
	return &pb.SwitchDetails{
		Name: string(s.Name),
		Arch: string(s.Arch),
		InstalledProgram: s.InstalledProgram,
	}
}

func ProgramDetailsToProto(pd *model.P4ProgramDetails) *pb.ProgramDetailsResponse {
	return &pb.ProgramDetailsResponse{
		ImplementedInterfaces: pd.ImplementedInterfaces,
	}
}
