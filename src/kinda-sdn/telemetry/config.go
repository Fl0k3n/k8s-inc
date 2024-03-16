package telemetry

import (
	"fmt"

	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

const ANY_PORT = -1
const ANY_IPv4 = "any"
const FULL_TELEMETRY_KEY = 0xFF00

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

func ActivateSource(ingressPort int) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_activate_source",
		Match: map[string]string{
			"standard_metadata.ingress_port": fmt.Sprintf("%d", ingressPort),
		},
		ActionName: "activate_source",
		ActionParams: map[string]string{},
	}
}

func ConfigureSource(
	srcAddr string, dstAddr string, srcPort int, dstPort int,
	maxHop int, hopMetadataLen int, insCnt int, insMask int, tunneled bool,
) connector.RawTableEntry {
	match := map[string]string{}
	ipv4TableName := "ipv4"
	sourceTableName := "tb_int_source"
	if tunneled {
		ipv4TableName = "nested_ipv4"
		sourceTableName = "tb_int_source_tunneled"
	}
	if srcAddr != ANY_IPv4 {
		match[fmt.Sprintf("hdr.%s.srcAddr", ipv4TableName)] = srcAddr
	}
	if dstAddr != ANY_IPv4 {
		match[fmt.Sprintf("hdr.%s.dstAddr", ipv4TableName)] = dstAddr
	}
	if srcPort != ANY_PORT {
		match["meta.layer34_metadata.l4_src"] = ExactPortTernary(srcPort)
	}
	if dstPort != ANY_PORT {
		match["meta.layer34_metadata.l4_dst"] = ExactPortTernary(dstPort)
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
		},
	}
}

func ExactIpv4Ternary(ip string) string {
	return fmt.Sprintf("%s&&&0xFFFFFFFF", ip)
}

func ExactPortTernary(port int) string {
	return fmt.Sprintf("%d&&&0xFFFF", port)
}

func ConfigureSink(egressPort int, sinkReportingPort int) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_int_sink",
		Match: map[string]string{
			"standard_metadata.egress_spec": fmt.Sprintf("%d", egressPort),
		},
		ActionName: "configure_sink",
		ActionParams: map[string]string{
			"sink_reporting_port": fmt.Sprintf("%d", sinkReportingPort),
		},
	}
}

// TODO allow having different collectors for different key, whatever that key might be
func Reporting(srcMac string, srcIp string,
	collectorMac string, collectorIp string, collectorUdpPort int,
) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_int_reporting",
		Match: map[string]string{},
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
