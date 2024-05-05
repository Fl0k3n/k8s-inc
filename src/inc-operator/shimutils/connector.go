package shimutils

import (
	"context"
	"errors"
	"sync"
	"time"

	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// thread-safe
type TelemetryPluginConnector struct {
	k8sClient client.Client
	sdnClient pbt.TelemetryServiceClient
	sdnConn *grpc.ClientConn
	shimNamespace string
	
	sdnClientInitLock sync.Mutex
	shimAccessDeadline time.Duration
	closed bool
}

func NewTelemetryPluginConnector(client client.Client) *TelemetryPluginConnector {
	return &TelemetryPluginConnector{
		k8sClient: client,
		sdnClient: nil,
		sdnConn: nil,
		shimNamespace: "",
		sdnClientInitLock: sync.Mutex{},
		shimAccessDeadline: time.Second * 2,
		closed: false,
	}
}

// error returned by consumer should indicate grpc layer issues, e.g. connection failure, not application layer errors
func (t *TelemetryPluginConnector) WithTelemetryClient(consumer func(client pbt.TelemetryServiceClient) error) error {
	t.sdnClientInitLock.Lock()
	if t.closed {
		t.sdnClientInitLock.Unlock()
		return errors.New("connector closed")
	}
	if t.sdnConn == nil {
		conn, err := t.getConnection()
		if err != nil {
			t.sdnClientInitLock.Unlock()
			return err
		}
		t.sdnConn = conn
		t.sdnClient = pbt.NewTelemetryServiceClient(t.sdnConn)
	}
	t.sdnClientInitLock.Unlock()
	// we allow concurrent close when someone accesses it
	err := consumer(t.sdnClient)
	if err != nil {
		go t.reset()
	}
	return err
}

func (t *TelemetryPluginConnector) GetNamespace() string {
	return t.shimNamespace
}

func (t *TelemetryPluginConnector) reset() {
	t.sdnClientInitLock.Lock()
	defer t.sdnClientInitLock.Unlock()
	if t.sdnConn != nil {
		t.sdnConn.Close()
		t.sdnConn = nil	
	}
}

// init lock must be held
func (t *TelemetryPluginConnector) getConnection() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.shimAccessDeadline)
	defer cancel()
	shim, err := LoadSDNShim(ctx, t.k8sClient)
	if err != nil {
		return nil, err
	}
	t.shimNamespace = shim.Namespace
	addr := shim.Spec.SdnConfig.TelemetryServiceGrpcAddr
	return grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (t *TelemetryPluginConnector) Close() {
	t.sdnClientInitLock.Lock()
	defer t.sdnClientInitLock.Unlock()
	t.closed = true
	if t.sdnConn != nil {
		t.sdnConn.Close()
		t.sdnConn = nil
	}
}
