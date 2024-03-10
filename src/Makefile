.PHONY: all
all:
	make --directory ./sdn-shim generate manifests install
	make --directory ./inc-operator generate manifests install
	kubectl apply -f sdn-shim/config/example/shim.yaml
	sleep 1
	kubectl apply -f inc-operator/config/example/eintdepl.yaml

