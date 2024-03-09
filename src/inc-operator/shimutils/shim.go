package shimutils

import (
	"context"
	"errors"

	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
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
