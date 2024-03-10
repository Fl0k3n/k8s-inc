package model

type TopologyGraph = map[DeviceName]Device

func TopologyToGraph(topo *Topology) TopologyGraph {
	G := map[DeviceName]Device{}
	for _, d := range topo.Devices {
		G[d.GetName()] = d
	}
	return G
}
