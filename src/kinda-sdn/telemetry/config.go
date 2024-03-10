package telemetry

import (
	"fmt"

	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

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
		Match: map[string]string{
			"hdr.ipv4.dstAddr": "0.0.0.0/1",
		},
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
	maxHop int, hopMetadataLen int, insCnt int, insMask int,
) connector.RawTableEntry {
	return connector.RawTableEntry{
		TableName: "tb_int_source",
		Match: map[string]string{
			"hdr.ipv4.srcAddr": srcAddr,
			"hdr.ipv4.dstAddr": dstAddr,
			"meta.layer34_metadata.l4_src": fmt.Sprintf("%d&&&0xFFFF", srcPort),
			"meta.layer34_metadata.l4_dst": fmt.Sprintf("%d&&&0xFFFF", dstPort),
		},
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

func AnyIpv4Ternary() string {
	return "0.0.0.0&&&0x0"
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
		Match: map[string]string{
			"hdr.ipv4.dstAddr": "0.0.0.0/1",
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
