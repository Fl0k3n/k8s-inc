package util

import (
	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	"google.golang.org/protobuf/encoding/prototext"
)

func UnmarshalP4Info(p4infoBytes []byte) (*p4_config_v1.P4Info, error) {
	p4Info := &p4_config_v1.P4Info{}
	err := prototext.Unmarshal(p4infoBytes, p4Info)
	return p4Info, err
}
