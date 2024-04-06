
SCHED_DIR=/home/flok3n/go/src/sigs.k8s.io/scheduler-plugins
SCHED_INT_CRD_DIR=$(SCHED_DIR)/pkg/intv1alpha
SCHED_SHIM_CRD_DIR=$(SCHED_DIR)/pkg/shimv1alpha

.PHONY: copy-crds-to-sched
copy-crds-to-sched:
	rm $(SCHED_INT_CRD_DIR)/*
	cp ./inc-operator/api/v1alpha1/* $(SCHED_INT_CRD_DIR)
	rm $(SCHED_SHIM_CRD_DIR)/*
	cp ./sdn-shim/api/v1alpha1/* $(SCHED_SHIM_CRD_DIR)

.PHONY: install
install:
	make --directory ./sdn-shim generate manifests install
	make --directory ./inc-operator generate manifests install
	make copy-crds-to-sched

.PHONY: all
all: install
	kubectl apply -f sdn-shim/config/example/shim.yaml
	#kubectl apply -f inc-operator/config/example/eintdepl.yaml
	
.PHONY: clean
clean:
	kubectl delete ExternalInNetworkTelemetryDeployments --all
	kubectl delete ExternalInNetworkTelemetryEndpoints --all
	kubectl delete deployments --all
	kubectl delete pods --all
	kubectl delete services --all

