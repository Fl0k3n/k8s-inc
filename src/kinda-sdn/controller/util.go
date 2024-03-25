package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
)

func getDevicesOfType[T model.Device](topo *model.Topology, deviceType model.DeviceType) []T {
	res := []T{}
	for _, node := range topo.Devices {
		if node.GetType() == deviceType {
			res = append(res, node.(T))
		}
	}	
	return res
}

func getIncSwitches(topo *model.Topology) []*model.IncSwitch {
	return getDevicesOfType[*model.IncSwitch](topo, model.INC_SWITCH)
}

func getExternalDevices(topo *model.Topology) []*model.ExternalDevice {
	return getDevicesOfType[*model.ExternalDevice](topo, model.EXTERNAL)
}

// decimal mask
func maskIp(ip string, mask int) string {
	if mask < 0 || mask > 32 {
		panic("Invalid mask")
	}
	if strings.Contains(ip, "/") {
		return ip
	}
	return fmt.Sprintf("%s/%d", ip, mask)
}

func getNetworkIp(ip string, mask int) string {
	parts := strings.Split(ip, ".")
	var ipv4 int32 = 0
	for i := 3; i >= 0; i-- {
		ipByte, err := strconv.ParseUint(parts[i], 10, 8)
		if err != nil {
			panic(err)
		}
		ipv4 += (int32(ipByte) << ((3 - i) * 8))
	}
	var maskBinary int32 = (1 << (32 - mask)) - 1
	ipv4 &= ^maskBinary
	for i := 3; i >= 0; i-- {
		b := ipv4 & 0xFF
		parts[i] = fmt.Sprintf("%d", b)
		ipv4 = ipv4 >> 8
	}
	return strings.Join(parts, ".")
}
