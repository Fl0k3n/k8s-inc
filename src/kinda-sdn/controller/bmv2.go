package controller

import (
	"context"
	"time"

	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

const DEVICE_ID = 0
const WRITE_ENTRY_TIMEOUT = 1 * time.Second

type Bmv2Manager struct {
	grpcAddr string
	conn *connector.P4RuntimeConnector
}

func NewBmv2Manager(grpcAddr string) *Bmv2Manager {
	return &Bmv2Manager{
		grpcAddr: grpcAddr,
		conn: nil,
	}
}

func (b *Bmv2Manager) convertEntry(entry *connector.RawTableEntry) (connector.TableEntry, error) {
	match, err := b.conn.BuildMatchEntry(entry.TableName, entry.Match)
	if err != nil {
		return connector.TableEntry{}, err
	}
	action, err := b.conn.BuildActionEntry(entry.ActionName, entry.ActionParams)
	if err != nil {
		return connector.TableEntry{}, err
	}
	return connector.TableEntry{
		TableName: entry.TableName,
		Match: match,
		Action: action,
	}, nil
}

func (b *Bmv2Manager) WriteInitialEntries(ctx context.Context, entries []connector.RawTableEntry) error {
	convertedEntries := make([]connector.TableEntry, len(entries))
	for i, entry := range entries {
		convertedEntry, err := b.convertEntry(&entry)
		if err != nil {
			return err
		}
		convertedEntries[i] = convertedEntry
	}
	for _, entry := range convertedEntries {
		if err := b.conn.WriteMatchActionEntry(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bmv2Manager) Open(ctx context.Context) error {
	conn := connector.NewP4RuntimeConnector(b.grpcAddr, DEVICE_ID)
	if err := conn.Open(ctx); err != nil {
		return err
	}
	b.conn = conn
	return nil
}

func (b *Bmv2Manager) InstallProgram(binPath string, p4InfoPath string) error {
	return b.conn.InstallProgram(context.Background(), binPath, p4InfoPath)
}

func (b *Bmv2Manager) Close() {
	if b.conn != nil {
		b.conn.Close()
	}
}
