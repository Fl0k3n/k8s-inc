package model

import "fmt"

type DeviceType = string
type DeviceName = string

const (
	NODE DeviceType = "node"
	INC_SWITCH DeviceType = "inc-switch"
	NET DeviceType = "net"
	EXTERNAL DeviceType = "external"
)

type IncSwitchArch string

const (
	BMv2 IncSwitchArch = "bmv2"
)

const NO_PROGRAM = "none"

type Link struct {
	To DeviceName
	MacAddr string
	Ipv4 string
	Mask int
}

type Device interface {
	GetName() DeviceName
	GetLinks() []*Link
	GetType() DeviceType
	GetPortNumberTo(DeviceName) (p int, ok bool)
	MustGetPortNumberTo(DeviceName) int
	MustGetLinkTo(DeviceName) *Link
	Equals(Device) bool
}

func NewLink(to DeviceName, mac string, ipv4 string, mask int) *Link {
	return &Link{
		To: to,
		MacAddr: mac,
		Ipv4: ipv4,
		Mask: mask,
	}
}

type BaseDevice struct {
	Name DeviceName
	Links []*Link
}

func (b *BaseDevice) GetName() DeviceName {
	return b.Name
}

func (b *BaseDevice) GetLinks() []*Link {
	return b.Links
}

// 0 based, increment if used in bmv2
func (b *BaseDevice) GetPortNumberTo(name DeviceName) (int, bool) {
	for i, link := range b.GetLinks() {
		if link.To == name {
			return i, true
		}
	}
	return 0, false
}

// 0 based, increment if used in bmv2
func (b *BaseDevice) MustGetPortNumberTo(name DeviceName) int {
	res, ok := b.GetPortNumberTo(name)
	if !ok {
		panic(fmt.Sprintf("Device %s doesn't have a link to %s", b.GetName(), name))
	}
	return res
}

func (b *BaseDevice) MustGetLinkTo(name DeviceName) *Link {
	return b.GetLinks()[b.MustGetPortNumberTo(name)]
}

func (b *BaseDevice) Equals(other Device) bool {
	if other == nil {
		return false
	}
	return b.Name == other.GetName()
}

type IncSwitch struct {
	BaseDevice
	Arch IncSwitchArch
	GrpcUrl string
	InstalledProgram string
	AllowClientProgrammability bool
}

func NewIncSwitch(name DeviceName, links []*Link, arch IncSwitchArch,
		grpcUrl string, installedProgram string, allowClientProgrammability bool) *IncSwitch {
	return &IncSwitch{
		BaseDevice: BaseDevice{
			Name: name,
			Links: links,
		},
		Arch: arch,
		GrpcUrl: grpcUrl,
		InstalledProgram: installedProgram,
		AllowClientProgrammability: allowClientProgrammability,
	}
}

func NewBmv2IncSwitch(name DeviceName, links []*Link, grpcUrl string) *IncSwitch {
	return NewIncSwitch(name, links, BMv2, grpcUrl, NO_PROGRAM, true)
}

func (i *IncSwitch) GetType() DeviceType {
	return INC_SWITCH
}

type Host struct {
	BaseDevice
}

func (h *Host) GetType() DeviceType {
	return NODE
}

type NetDevice struct {
	BaseDevice
}

func (n *NetDevice) GetType() DeviceType {
	return NET
}

type ExternalDevice struct {
	BaseDevice
}

func (e *ExternalDevice) GetType() DeviceType {
	return EXTERNAL
}

