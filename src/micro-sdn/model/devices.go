package model

type DeviceType string
type DeviceName string

const (
	NODE DeviceType = "node"
	INC_SWITCH DeviceType = "inc-switch"
	NET DeviceType = "net"
	EXTERNAL DeviceType = "external"
)

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

type IncSwitch struct {
	BaseDevice
	GrpcUrl string
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

