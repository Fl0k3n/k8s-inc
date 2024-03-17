package utils

import (
	"fmt"
	"slices"

	"github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
)

type TelemetryRequestType string

const (
	INGRESS_TO_EXTERNAL TelemetryRequestType = "ie"
	EXTERNAL_TO_INGRESS TelemetryRequestType = "ei"
	INGRESS_TO_PODS     TelemetryRequestType = "ip"	
	PODS_TO_INGRESS     TelemetryRequestType = "pi"
)

func BuildCollectionName(baseName string, reqType TelemetryRequestType) string {
	return fmt.Sprintf("%s-%s", baseName, string(reqType))
}

func IngressInfoChanged(desired v1alpha1.IngressInfo, actual *v1alpha1.IngressInfo) bool {
	return actual == nil || desired.IngressType != actual.IngressType || !slices.Equal(desired.NodeNames, actual.NodeNames)
}

func MonitoringPolicyChanged(desired v1alpha1.MonitoringPolicy, actual *v1alpha1.MonitoringPolicy) bool {
	return actual == nil || desired != *actual
}
