package telemetry

import (
	"fmt"

	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

const ANY_PORT = -1
const ANY_IPv4 = "any"
const FULL_TELEMETRY_KEY = 0xFF00

type MatchIdentifier = string

type MatchKey interface {
	ToIdentifier() MatchIdentifier
}

type ConfigureTransitKey struct {

}

func (c *ConfigureTransitKey) ToIdentifier() MatchIdentifier {
	return ""
}

type ConfigureSinkKey struct {
	egressPort int
}

func (c *ConfigureSinkKey) ToIdentifier() MatchIdentifier {
	return fmt.Sprintf("%d", c.egressPort)
}

type ReportingKey struct {
	collectionId int
}

func (c *ReportingKey) ToIdentifier() MatchIdentifier {
	return fmt.Sprintf("%d", c.collectionId)
}

type ActivateSourceKey struct {
	ingressPort int
}

func (c *ActivateSourceKey) ToIdentifier() MatchIdentifier {
	return fmt.Sprintf("%d", c.ingressPort)
}

type ConfigureSourceKey struct {
	srcAddr string
	dstAddr string
	srcPort int
	dstPort int
	tunneled bool
}

func (c *ConfigureSourceKey) ToIdentifier() string {
	isTunneled := "tun"
	if !c.tunneled {
		isTunneled = "raw"
	}
	return fmt.Sprintf("%s_%s_%d_%d_%s", c.srcAddr, c.dstAddr, c.srcPort, c.dstPort, isTunneled)
}

func Forward(ip string, srcMac string, dstMac string, port string) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "ingress.Forward.ipv4_lpm",
		Match: map[string]string{
			"hdr.ipv4.dstAddr": ip,
		},
		ActionName: "ingress.Forward.ipv4_forward",
		ActionParams: map[string]string{
			"srcAddr": srcMac,
			"dstAddr": dstMac,
			"port": port,
		},
	}
}

func DefaultRoute(srcMac string, dstMac string, port string) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "ingress.Forward.ipv4_lpm",
		Match: map[string]string{},
		ActionName: "ingress.Forward.ipv4_forward",
		ActionParams: map[string]string{
			"srcAddr": srcMac,
			"dstAddr": dstMac,
			"port": port,
		},
	}
}

func Arp(ip string, mac string) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "ingress.Forward.arp_exact",
		Match: map[string]string{
			"hdr.arp.dstIp": ip,
		},
		ActionName: "ingress.Forward.reply_arp",
		ActionParams: map[string]string{
			"targetMac": mac,
		},
	}
}

func Transit(switchId int, mtu int) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_int_transit",
		Match: map[string]string{},
		ActionName: "configure_transit",
		ActionParams: map[string]string{
			"switch_id": fmt.Sprintf("%d", switchId),
			"l3_mtu": fmt.Sprintf("%d", mtu),
		},
	}
}

func ActivateSource(key *ActivateSourceKey) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_activate_source",
		Match: map[string]string{
			"standard_metadata.ingress_port": fmt.Sprintf("%d", key.ingressPort),
		},
		ActionName: "activate_source",
		ActionParams: map[string]string{},
	}
}

func ConfigureSource(
	key *ConfigureSourceKey,
	maxHop int, hopMetadataLen int, insCnt int, insMask int, collectionId int,
) connector.RawTableEntry {
	match := map[string]string{}
	ipv4TableName := "ipv4"
	sourceTableName := "tb_int_source"
	if key.tunneled {
		ipv4TableName = "nested_ipv4"
		sourceTableName = "tb_int_source_tunneled"
	}
	if key.srcAddr != ANY_IPv4 {
		match[fmt.Sprintf("hdr.%s.srcAddr", ipv4TableName)] = key.srcAddr
	}
	if key.dstAddr != ANY_IPv4 {
		match[fmt.Sprintf("hdr.%s.dstAddr", ipv4TableName)] = key.dstAddr
	}
	if key.srcPort != ANY_PORT {
		match["meta.layer34_metadata.l4_src"] = ExactPortTernary(key.srcPort)
	}
	if key.dstPort != ANY_PORT {
		match["meta.layer34_metadata.l4_dst"] = ExactPortTernary(key.dstPort)
	}
	return connector.RawTableEntry{
		TableName: sourceTableName,
		Match: match,
		ActionName: "configure_source",
		ActionParams: map[string]string{
			"max_hop": fmt.Sprintf("%d", maxHop),
			"hop_metadata_len": fmt.Sprintf("%d", hopMetadataLen),
			"ins_cnt": fmt.Sprintf("%d", insCnt),
			"ins_mask": fmt.Sprintf("%d", insMask),
			"collection_id": fmt.Sprintf("%d", collectionId),
		},
	}
}

func ExactIpv4Ternary(ip string) string {
	return fmt.Sprintf("%s&&&0xFFFFFFFF", ip)
}

func ExactPortTernary(port int) string {
	return fmt.Sprintf("%d&&&0xFFFF", port)
}

func ConfigureSink(key *ConfigureSinkKey, sinkReportingPort int) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_int_sink",
		Match: map[string]string{
			"standard_metadata.egress_spec": fmt.Sprintf("%d", key.egressPort),
		},
		ActionName: "configure_sink",
		ActionParams: map[string]string{
			"sink_reporting_port": fmt.Sprintf("%d", sinkReportingPort),
		},
	}
}

func Reporting(
	key *ReportingKey,
	srcMac string, srcIp string,
	collectorMac string, collectorIp string, collectorUdpPort int,
) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_int_reporting",
		Match: map[string]string{
			"hdr.int_header.collection_id": fmt.Sprintf("%d", key.collectionId),
		},
		ActionName: "send_report",
		ActionParams: map[string]string{
			"dp_mac": srcMac,
			"dp_ip": srcIp, 
			"collector_mac": collectorMac,
			"collector_ip": collectorIp,
			"collector_port": fmt.Sprintf("%d", collectorUdpPort),
		},
	}
}
