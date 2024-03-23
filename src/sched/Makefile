
ifdef GOPATH
SCHED_PATH:=$(GOPATH)/src/sigs.k8s.io/scheduler-plugins
else
SCHED_PATH:=$(HOME)/go/src/sigs.k8s.io/scheduler-plugins
endif

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

SCHED_MAKE=make --directory $(SCHED_PATH)

IMAGE_NAME=localhost:5000/scheduler-plugins/kube-scheduler:1.0.0

CONTROL_PLANE_CONTAINER_ID:=$(shell docker ps | grep control-plane | awk '{print $$1}')

CLUSTER_NAME:=$(shell kind get clusters)

.PHONY: build
build:
	$(SCHED_MAKE) local-image
	-docker rmi $(IMAGE_NAME)
	docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest $(IMAGE_NAME)

.PHONY: install
install:
	test ! -z $(CONTROL_PLANE_CONTAINER_ID)
	kind load docker-image $(IMAGE_NAME) --name $(CLUSTER_NAME)
	docker cp $(ROOT_DIR)/config/kube-scheduler-conf.yaml $(CONTROL_PLANE_CONTAINER_ID):/etc/kubernetes/kube-scheduler-conf.yaml
	docker cp $(ROOT_DIR)/config/kube-scheduler.yaml $(CONTROL_PLANE_CONTAINER_ID):/etc/kubernetes/manifests/kube-scheduler.yaml
	docker exec $(CONTROL_PLANE_CONTAINER_ID) systemctl restart kubelet.service

.PHONY: all
all: build install

.PHONY: copy-to-scheduler
copy-to-scheduler:
	rm -rf $(SCHED_PATH)/pkg/internaltelemetry
	cp -r $(ROOT_DIR)/plugins/internaltelemetry $(SCHED_PATH)/pkg/internaltelemetry
