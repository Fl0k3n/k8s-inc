package controller

import (
	"github.com/Fl0k3n/k8s-inc/micro-sdn/model"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
)

func LinkToDao(link *model.Link) *pb.Link {
	return &pb.Link{
		PeerName: string(link.To),
	}
}

func convertDeviceType(deviceType model.DeviceType) pb.DeviceType {
	var dt pb.DeviceType
	switch deviceType {
	case model.NET:
		dt = pb.DeviceType_NET
	case model.EXTERNAL:
		dt = pb.DeviceType_EXTERNAL
	case model.INC_SWITCH:
		dt = pb.DeviceType_INC_SWITCH
	case model.NODE:
		dt = pb.DeviceType_HOST
	}
	return dt
}

func DeviceToDao(device model.Device) *pb.Device {
	links := make([]*pb.Link, len(device.GetLinks()))
	for i, link := range device.GetLinks() {
		links[i] = LinkToDao(link)
	}
	return &pb.Device{
		Name: string(device.GetName()),
		DeviceType:  convertDeviceType(device.GetType()),
		Links: links,
	}
}

func TopologyModelToDao(topo *model.Topology) *pb.TopologyResponse {
	devices := make([]*pb.Device, len(topo.Devices))
	for i, dev := range topo.Devices {
		devices[i] = DeviceToDao(dev)
	}
	return &pb.TopologyResponse{
		Graph: devices,
	}
}
