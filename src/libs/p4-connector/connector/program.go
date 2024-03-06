package connector

import (
	"context"
	"os"

	"github.com/Fl0k3n/k8s-inc/libs/p4r/util"
)


func (p *P4RuntimeConnector) InstallProgram(ctx context.Context, binPath string, p4infoPath string) error {
	res, err := p.p4rtc.SetFwdPipe(ctx, binPath, p4infoPath, 0)
	p.p4info = res.P4Info
	return err
}

func (p *P4RuntimeConnector) UseInstalledProgram(p4infoPath string) error {
	p4infoBytes, err := os.ReadFile(p4infoPath)
	if err != nil {
		return err
	}
	p4Info, err := util.UnmarshalP4Info(p4infoBytes)
	if err != nil {
		return err
	}
	p.p4info = p4Info
	p.p4rtc.SetLocalP4Info(p4Info)
	return nil
}
