package shimutils

import (
	"context"
	"errors"

	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)


func LoadSDNShim(ctx context.Context, client client.Client) (*shimv1alpha1.SDNShim, error) {
	shims := &shimv1alpha1.SDNShimList{}
	if err := client.List(ctx, shims); err != nil {
		return nil, err
	}
	if len(shims.Items) == 0 {
		return nil, errors.New("SDNShim unavailable")
	}
	return &shims.Items[0], nil
}


func LoadTopology(ctx context.Context, client client.Client) (*shimv1alpha1.Topology, error) {
	topos := &shimv1alpha1.TopologyList{} // TODO query just 1
	if err := client.List(ctx, topos); err != nil {
		return nil, err
	}
	if len(topos.Items) == 0 {
		return nil, errors.New("topology unavailable")
	}
	return &topos.Items[0], nil
}

func LoadNodes(ctx context.Context, client client.Client) (map[string]*v1.Node, error) {
	nodes := &v1.NodeList{}
	if err := client.List(ctx, nodes); err != nil {
		return nil, err
	}
	res := map[string]*v1.Node{}
	for _, n := range nodes.Items {
		res[n.Name] = &n
	}
	return res, nil
}

func LoadSwitches(ctx context.Context, client client.Client) (map[string]*shimv1alpha1.IncSwitch, error) {
	switches := &shimv1alpha1.IncSwitchList{}
	if err := client.List(ctx, switches); err != nil {
		return nil, err
	}
	res := map[string]*shimv1alpha1.IncSwitch{}
	for _, s := range switches.Items {
		res[s.Name] = &s
	}
	return res, nil
}
