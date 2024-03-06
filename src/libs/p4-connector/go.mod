module github.com/Fl0k3n/k8s-inc/libs/p4-connector

go 1.21.7

require (
	github.com/Fl0k3n/k8s-inc/libs/p4r v0.0.0
	github.com/p4lang/p4runtime v1.4.0-rc.5
	google.golang.org/grpc v1.58.3
	sigs.k8s.io/controller-runtime v0.17.2
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240304212257-790db918fca8 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/Fl0k3n/k8s-inc/libs/p4r => ../p4r
