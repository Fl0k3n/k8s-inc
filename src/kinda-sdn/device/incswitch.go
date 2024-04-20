package device

import (
	"context"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

// thread-safe
type IncSwitch interface {
	GetArch() model.IncSwitchArch
	WriteEntry(context.Context, connector.RawTableEntry) error
	DeleteEntry(context.Context, connector.RawTableEntry) error
}
