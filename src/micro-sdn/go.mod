module github.com/Fl0k3n/k8s-inc/micro-sdn

require github.com/Fl0k3n/k8s-inc/proto v0.0.1

require (
	github.com/golang/protobuf v1.5.3 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
)

replace github.com/Fl0k3n/k8s-inc/proto => ../proto/go

go 1.21.7
