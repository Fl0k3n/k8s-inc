package controller

import (
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/graphutils"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
	sets "github.com/hashicorp/go-set"
)

func (k *KindaSdn) buildBasicForwardingEntries() map[model.DeviceName][]connector.RawTableEntry {
	// for now we configure only Bmv2 switches
	// default routes are configured to first found external device (if any)
	// this function assumes that G is a tree
	G := model.TopologyToGraph(k.topo)
	incSwitches := getIncSwitches(k.topo)
	res := map[model.DeviceName][]connector.RawTableEntry{}
	for _, incSwitch := range incSwitches {
		entries := k.createArpEntries(incSwitch)
		entries = append(entries, k.createDirectForwardEntries(incSwitch, G)...)
		nextHops := graphutils.GetNextHopsMap(G, incSwitch)
		entries = append(entries, k.createIndirectForwardEntries(incSwitch, G, nextHops)...)
		res[incSwitch.Name] = entries
	}
	return res
}

func (k *KindaSdn) createArpEntries(incSwitch *model.IncSwitch) []connector.RawTableEntry {
	res := []connector.RawTableEntry{}
	for _, link := range incSwitch.Links {
		res = append(res, k.p4Delegate(incSwitch).GetArpEntry(link.Ipv4, link.MacAddr))	
	}
	return res
}

func (k *KindaSdn) createDirectForwardEntries(incSwitch *model.IncSwitch, G model.TopologyGraph) []connector.RawTableEntry {
	res := []connector.RawTableEntry{}
	for _, link := range incSwitch.Links {
		peerPort := incSwitch.MustGetPortNumberTo(link.To) + 1
		peerMac := G[link.To].MustGetLinkTo(incSwitch.Name).MacAddr
		entry := k.p4Delegate(incSwitch).GetForwardEntry(maskIp(link.Ipv4, 32), link.MacAddr, peerMac, peerPort)
		res = append(res, entry)
	}
	return res
}

func (k *KindaSdn) createIndirectForwardEntries(
	incSwitch *model.IncSwitch,
	G model.TopologyGraph,
	nextHops map[model.DeviceName]model.Device,
) []connector.RawTableEntry {
	res := []connector.RawTableEntry{}
	externalDevices := getExternalDevices(k.topo)
	if len(externalDevices) > 0 {
		defaultRouteDevice := externalDevices[0]
		nextHop := nextHops[defaultRouteDevice.Name]
		srcPort := incSwitch.MustGetPortNumberTo(nextHop.GetName())
		dstMac := nextHop.MustGetLinkTo(incSwitch.Name).MacAddr

		entry := k.p4Delegate(incSwitch).GetDefaultRouteEntry(incSwitch.Links[srcPort].MacAddr, dstMac, srcPort+1)
		// TODO SKIPPING DEFAULT BECAUSE IT CAUSES DUPLICATE PACKETS IN TELEMETRY PROGRAM FOR WHATEVER REASON
		_ = entry
	}
	addedNetworks := sets.New[string](0)
	for devName, dev := range G {
		// skip self and direct link
		nextHop := nextHops[devName]
		if nextHop.GetName() == devName {
			continue
		}
		srcPort := incSwitch.MustGetPortNumberTo(nextHop.GetName())
		srcMac := incSwitch.GetLinks()[srcPort].MacAddr
		dstMac := nextHop.MustGetLinkTo(incSwitch.Name).MacAddr
		for _, link := range dev.GetLinks() {
			networkIp := maskIp(getNetworkIp(link.Ipv4, link.Mask), link.Mask)
			if !addedNetworks.Contains(networkIp) {
				addedNetworks.Insert(networkIp)
				entry := k.p4Delegate(incSwitch).GetForwardEntry(
					networkIp, srcMac, dstMac, srcPort + 1,
				)
				res = append(res, entry)
			}
		}
	}
	return res
}
