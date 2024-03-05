package client

import "operators/libs/p4r/util/conversion"


type Port struct {
	bytes []byte
	i     uint32
}

func NewPortFromInt(i uint32) Port {
	b, _ := conversion.UInt32ToBinaryCompressed(i)
	return Port{b, i}
}

func NewPort(bytes []byte) Port {
	return Port{bytes, 0xffffffff}
}

func (p Port) AsBytes() []byte {
	return p.bytes
}
