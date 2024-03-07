package controller

import (
	"github.com/Fl0k3n/k8s-inc/micro-sdn/model"
	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
)

type MicroSdn struct {
	pb.UnimplementedSdnFrontendServer
	topo *model.Topology
}

func NewMicroSdn(topo *model.Topology) *MicroSdn {
	return &MicroSdn{
		topo: topo,
	}
}



