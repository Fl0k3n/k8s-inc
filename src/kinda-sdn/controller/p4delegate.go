package controller

import (
	"fmt"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/telemetry"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)


type P4Delegate interface {
	GetArpEntry(reqIp string, respMac string) connector.RawTableEntry
	GetForwardEntry(maskedIp string, srcMac string, dstMac string, port int) connector.RawTableEntry
	GetDefaultRouteEntry(srcMac string, dstMac string, port int) connector.RawTableEntry
}

func (k *KindaSdn) p4Delegate(incSwitch *model.IncSwitch) P4Delegate {
	switch incSwitch.InstalledProgram {
	case telemetry.PROGRAM_INTERFACE:
		return k.telemetryService
	default:
		panic(fmt.Sprintf("Couldn't find p4 delegate for program %s", incSwitch.InstalledProgram))
	}
}
