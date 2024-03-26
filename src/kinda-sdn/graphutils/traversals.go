package graphutils

import (
	"slices"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	deques "github.com/gammazero/deque"
	sets "github.com/hashicorp/go-set"
)

type ParentsMap = map[model.DeviceName]model.Device

func Bfs(G model.TopologyGraph, source model.Device) ParentsMap {
	visited := sets.New[string](0)
	queue := deques.New[model.Device]()
	queue.PushBack(source)
	visited.Insert(source.GetName())
	parents := map[model.DeviceName]model.Device{}
	parents[source.GetName()] = nil

	for queue.Len() > 0 {
		cur := queue.PopFront()
		for _, link := range cur.GetLinks() {
			if !visited.Contains(link.To) {
				visited.Insert(link.To)
				queue.PushBack(G[link.To])
				parents[link.To] = cur
			}
		}
	}
	return parents
}

func BfsPathFind(G model.TopologyGraph, source model.Device, target model.Device) []string {
	parents := Bfs(G, source)
	if _, ok := parents[target.GetName()]; !ok {
		panic("No path")
	}
	path := []string{}
	cur := target
	for cur != nil {
		path = append(path, cur.GetName())
		parent := parents[cur.GetName()]
		cur = parent
	}
	slices.Reverse(path)
	return path
}

// returns next hop on a closest path (in terms of number of links)
func GetNextHopsMap(G model.TopologyGraph, source model.Device) map[model.DeviceName]model.Device {
	parents := Bfs(G, source)
	res := map[model.DeviceName]model.Device{}
	res[source.GetName()] = source

	for devName := range G {
		if _, ok := res[devName]; ok {
			continue
		}
		cur := devName
		path := []string{cur}	
		var nextHop string
		for {
			parent := parents[cur]	
			if parent.Equals(source) {
				nextHop = cur
				break
			}
			if next, ok := res[cur]; ok {
				nextHop = next.GetName()
				break
			}
			cur = parent.GetName()
			path = append(path, cur)	
		}
		for _, dev := range path {
			res[dev] = G[nextHop]
		}
	}
	return res
}

// returns next hop on a closest path (in terms of number of links)
func FindNextHop(G model.TopologyGraph, source model.Device, target model.Device) model.Device {
	path := BfsPathFind(G, source, target)
	nextHop := path[1]
	return G[nextHop]
}

