package controller

import (
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/programs"
)

func (k *KindaSdn) p4Delegate(incSwitch *model.IncSwitch) programs.P4Delegate {
	return k.programRegistry.MustFindDelegate(incSwitch.InstalledProgram)
}
