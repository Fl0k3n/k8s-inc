package shimutils

import shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"

type TopologyGraph = map[string]shimv1alpha1.NetworkDevice

func TopologyToGraph(topo *shimv1alpha1.Topology) TopologyGraph {
	res := map[string]shimv1alpha1.NetworkDevice{}
	for _, dev := range topo.Spec.Graph {
		res[dev.Name] = dev
	}
	return res
}
