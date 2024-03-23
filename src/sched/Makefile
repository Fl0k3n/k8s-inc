
ifdef GOPATH
SCHED_PATH:=$(GOPATH)/src/sigs.k8s.io/scheduler-plugins
else
SCHED_PATH:=$(HOME)/go/src/sigs.k8s.io/scheduler-plugins
endif

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

SCHED_MAKE=make --directory $(SCHED_PATH)

IMAGE_NAME=localhost:5000/scheduler-plugins/kube-scheduler:1.0.0

CONTROL_PLANE_CONTAINER_ID:=$(shell docker ps | grep control-plane | awk '{print $$1}')
CONTROL_PLANE_IP:=$(shell docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(CONTROL_PLANE_CONTAINER_ID))

CLUSTER_NAME:=$(shell kind get clusters)

.PHONY: build
build:
	$(SCHED_MAKE) local-image
	-docker rmi $(IMAGE_NAME)
	docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest $(IMAGE_NAME)

.PHONY: install
install:
	test ! -z $(CONTROL_PLANE_CONTAINER_ID)
	-docker exec $(CONTROL_PLANE_CONTAINER_ID) crictl rmi $(IMAGE_NAME)
	make load-image
	docker cp $(ROOT_DIR)/config/kube-scheduler-conf.yaml $(CONTROL_PLANE_CONTAINER_ID):/etc/kubernetes/kube-scheduler-conf.yaml
	docker cp $(ROOT_DIR)/config/kube-scheduler.yaml $(CONTROL_PLANE_CONTAINER_ID):/etc/kubernetes/manifests/kube-scheduler.yaml
	docker exec $(CONTROL_PLANE_CONTAINER_ID) systemctl restart kubelet.service
	make restart

.PHONY: all
all: build install

# ---- debugging

.PHONY: build-debug
build-debug:
	$(SCHED_MAKE) local-image-debug
	-docker rmi $(IMAGE_NAME)
	docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest $(IMAGE_NAME)

.PHONY: install-debug
install-debug:
	test ! -z $(CONTROL_PLANE_CONTAINER_ID)
	-docker exec $(CONTROL_PLANE_CONTAINER_ID) crictl rmi $(IMAGE_NAME)
	make load-image
	docker cp $(ROOT_DIR)/config/kube-scheduler-conf.yaml $(CONTROL_PLANE_CONTAINER_ID):/etc/kubernetes/kube-scheduler-conf.yaml
	docker cp $(ROOT_DIR)/config/kube-scheduler-debug.yaml $(CONTROL_PLANE_CONTAINER_ID):/etc/kubernetes/manifests/kube-scheduler.yaml
	make restart

.PHONY: debug-info
debug-info:
	echo "Control Plane IP: $(CONTROL_PLANE_IP)"

.PHONY: all-debug
all-debug: build-debug install-debug


# ---- utils

.PHONY: load-image
load-image:
	docker save -o $(ROOT_DIR)/img.tar $(IMAGE_NAME)
	docker cp $(ROOT_DIR)/img.tar $(CONTROL_PLANE_CONTAINER_ID):/img.tar
	docker exec $(CONTROL_PLANE_CONTAINER_ID) ctr --namespace k8s.io image import /img.tar
	docker exec $(CONTROL_PLANE_CONTAINER_ID) rm /img.tar
	rm $(ROOT_DIR)/img.tar

.PHONY: restart
restart:
	$(ROOT_DIR)/restart-sched-pod.sh $(CLUSTER_NAME) $(CONTROL_PLANE_CONTAINER_ID)


.PHONY: update-scheduler-config
update-scheduler-config:
	docker cp $(ROOT_DIR)/config/kube-scheduler-conf.yaml $(CONTROL_PLANE_CONTAINER_ID):/etc/kubernetes/kube-scheduler-conf.yaml
	make restart
