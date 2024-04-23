module github.com/Fl0k3n/k8s-inc/kinda-sdn

require github.com/Fl0k3n/k8s-inc/proto v0.0.1

require (
	github.com/gammazero/deque v0.2.1
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/hashicorp/go-set v0.1.14
	github.com/p4lang/p4runtime v1.4.0-rc.5 // indirect
	sigs.k8s.io/controller-runtime v0.17.2 // indirect
)

require (
	github.com/Fl0k3n/k8s-inc/libs/p4-connector v0.0.0
	github.com/Fl0k3n/k8s-inc/libs/p4r v0.0.0 // indirect
	github.com/golang/protobuf v1.5.3
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240304212257-790db918fca8 // indirect
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
)

replace github.com/Fl0k3n/k8s-inc/proto => ../proto/go

replace github.com/Fl0k3n/k8s-inc/libs/p4-connector => ../libs/p4-connector

replace github.com/Fl0k3n/k8s-inc/libs/p4r => ../libs/p4r

go 1.21.7
