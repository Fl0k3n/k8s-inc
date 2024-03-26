package programs

import "github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"

type P4Delegate interface {
	GetArpEntry(reqIp string, respMac string) connector.RawTableEntry
	GetForwardEntry(maskedIp string, srcMac string, dstMac string, port int) connector.RawTableEntry
	GetDefaultRouteEntry(srcMac string, dstMac string, port int) connector.RawTableEntry
}
