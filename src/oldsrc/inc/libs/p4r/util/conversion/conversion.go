package conversion

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

func IpToBinary(ipStr string) ([]byte, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("not a valid IP: %s", ipStr)
	}
	return []byte(ip.To4()), nil
}

func MacToBinary(macStr string) ([]byte, error) {
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return nil, err
	}
	return []byte(mac), nil
}

// bigendian
func StringToBinaryUint(str string, bitwidth int) ([]byte, error) {
	u, err := strconv.ParseUint(str, 0, bitwidth)
	if err != nil {
		return nil, err
	}
	return NumberToBinaryUint(u, bitwidth), nil
}

// bigendian
func NumberToBinaryUint(num uint64, bitwidth int) []byte {
	numBytes := bitwidth / 8
	if bitwidth % 8 != 0 {
		numBytes += 1
	}
	res := make([]byte, numBytes)
	for i := numBytes - 1; i >= 0; i-- {
		res[i] = byte((num >> (8 * (numBytes - 1 - i))) & 0xFF)
	}
	return res
}

func UInt32ToBinary(i uint32, numBytes int) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b[numBytes:], nil
}

func UInt32ToBinaryCompressed(i uint32) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	for idx := 0; idx < 4; idx++ {
		if b[idx] != 0 {
			return b[idx:], nil
		}
	}
	return []byte{'\x00'}, nil
}

func ToCanonicalBytestring(bytes []byte) []byte {
	if len(bytes) == 0 {
		return bytes
	}
	i := 0
	for _, b := range bytes {
		if b != 0 {
			break
		}
		i++
	}
	if i == len(bytes) {
		return bytes[:1]
	}
	return bytes[i:]
}
