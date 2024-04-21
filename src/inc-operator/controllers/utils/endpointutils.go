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

// TODO: remember about namespaces
func BuildIntentId(baseName string, reqType TelemetryRequestType) string {
	return fmt.Sprintf("%s-%s", baseName, string(reqType))
}

func BuildCollectionId(baseName string) string {
	return fmt.Sprintf("external-%s", baseName)
}

func IngressInfoChanged(desired v1alpha1.IngressInfo, actual *v1alpha1.IngressInfo) bool {
	return actual == nil || desired.IngressType != actual.IngressType || !slices.Equal(desired.NodeNames, actual.NodeNames)
}

func MonitoringPolicyChanged(desired v1alpha1.MonitoringPolicy, actual *v1alpha1.MonitoringPolicy) bool {
	return actual == nil || desired != *actual
}

func BuildInternalIntentId(baseName string, sourceDeplName string, targetDeplName string) string {
	return fmt.Sprintf("%s-%s-%s", baseName, sourceDeplName, targetDeplName)
}

func BuildInternalCollectionId(baseName string) string {
	return fmt.Sprintf("internal-%s", baseName)
}

type DeploymentPair struct {First, Second string}

func PairedDeployments(endpoints *v1alpha1.InternalInNetworkTelemetryEndpoints) []DeploymentPair {
	if len(endpoints.Spec.DeploymentEndpoints) != 2 {
		panic("expected 2 endpoints")
	}
	depl1 := endpoints.Spec.DeploymentEndpoints[0].DeploymentName
	depl2 := endpoints.Spec.DeploymentEndpoints[1].DeploymentName
	return []DeploymentPair{{depl1, depl2}, {depl2, depl1}}
}
