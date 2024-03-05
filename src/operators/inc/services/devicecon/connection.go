package devicecon

import (
	"context"
	"fmt"
	"operators/libs/p4r/client"

	p4_config_v1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4_v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/log"
)


type P4RuntimeConnector struct {
	deviceAddr string
	deviceID uint64
	conn *grpc.ClientConn
	p4rtc *client.Client
	p4info *p4_config_v1.P4Info
}

func NewP4RuntimeConnector(deviceAddr string, deviceID uint64) *P4RuntimeConnector {
	return &P4RuntimeConnector{
		deviceAddr: deviceAddr,
		deviceID: deviceID,
		conn: nil,
		p4rtc: nil,
		p4info: nil,
	}
}

func handleStreamMessages(ctx context.Context, messageCh <-chan *p4_v1.StreamMessageResponse) {
	log := log.FromContext(ctx)
	for message := range messageCh {
		switch m := message.Update.(type) {
		case *p4_v1.StreamMessageResponse_Packet:
			log.Info("Received PacketIn")
		case *p4_v1.StreamMessageResponse_Digest:
			log.Info("Received DigestList")
		case *p4_v1.StreamMessageResponse_IdleTimeoutNotification:
			log.Info("Received IdleTimeoutNotification")
		case *p4_v1.StreamMessageResponse_Error:
			log.Info("Received StreamError")
			log.Info(m.Error.Message)
		default:
			log.Info("Received unknown stream message")
		}
	}
}

// establishes connection and blocks until caller is the primary client or context is cancelled
func (p *P4RuntimeConnector) Open(ctx context.Context) error {
	log := log.FromContext(ctx)
	// TODO creds
	log.Info("hellooooooo")
	conn, err := grpc.Dial(p.deviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error(err, "Failed to establish grpc connection")
		return err
	}
	p.conn = conn
	c := p4_v1.NewP4RuntimeClient(conn)
	resp, err := c.Capabilities(ctx, &p4_v1.CapabilitiesRequest{})
	if err != nil {
		log.Error(err, "Error in Capabilities RPC")
		return err
	}
	log.Info(fmt.Sprintf("P4Runtime server version is %s", resp.P4RuntimeApiVersion))

	electionID := &p4_v1.Uint128{High: 0, Low: 1}
	messageCh := make(chan *p4_v1.StreamMessageResponse, 1000)

	p.p4rtc = client.NewClient(c, p.deviceID, electionID)
	arbitrationCh := make(chan bool)
	go p.p4rtc.Run(ctx, arbitrationCh, messageCh)
	go handleStreamMessages(ctx, messageCh)

	waitCh := make(chan struct{})

	go func() {
		sent := false
		for isPrimary := range arbitrationCh {
			if isPrimary {
				log.Info("We are the primary client!")
				if !sent {
					waitCh <- struct{}{}
					sent = true
				}
			} else {
				log.Info("We are not the primary client!")
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCh:
		return nil
	}
}

func (p *P4RuntimeConnector) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}
