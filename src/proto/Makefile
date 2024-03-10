CC = protoc --go_out=go --go_opt=paths=source_relative --go-grpc_out=go --go-grpc_opt=paths=source_relative 

.PHONY: sdn
sdn: $(wildcard sdn/*.proto)
	$(CC) sdn/sdn.proto

.PHONY: telemetry
telemetry: $(wildcard sdn/telemetry/*.proto)
	$(CC) sdn/telemetry/telemetry.proto

.PHONY: all
all: sdn telemetry
	echo "Built all protos"
