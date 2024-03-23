#!/bin/bash

CLUSTER_NAME=$1
CONTROL_PLANE_CONTAINER_ID=$2

RM_CMD="crictl ps | grep kube-scheduler-${CLUSTER_NAME}-control-plane | awk '{print \$1}' | xargs crictl rm -f 1>/dev/null"

# twice to make sure that new is definitely picked up
docker exec ${CONTROL_PLANE_CONTAINER_ID} /bin/bash -c "${RM_CMD}"
docker exec ${CONTROL_PLANE_CONTAINER_ID} systemctl restart kubelet.service
docker exec ${CONTROL_PLANE_CONTAINER_ID} /bin/bash -c "${RM_CMD}"
