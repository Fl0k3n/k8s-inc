/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const INTERNAL_TELEMETRY_SCHEDULER_NAME = "internal-telemetry"
const INTERNAL_TELEMETRY_POD_DEPLOYMENT_OWNER_LABEL = "inc.kntp.com/owned-by-iintdepl"
const INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL = "inc.kntp.com/part-of-deployment"

// InternalInNetworkTelemetryDeploymentReconciler reconciles a InternalInNetworkTelemetryDeployment object
type InternalInNetworkTelemetryDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetrydeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetrydeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetrydeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=deployments/finalizers,verbs=update

func (r *InternalInNetworkTelemetryDeploymentReconciler) getDeploymentFullName(
	intdeplName string,
	partialDeploymentName string,
) string {
	return fmt.Sprintf("%s-%s", intdeplName, partialDeploymentName)
}

func (r *InternalInNetworkTelemetryDeploymentReconciler) reconcileDeployment(
	ctx context.Context,
	intdepl *incv1alpha1.InternalInNetworkTelemetryDeployment,
	deployTemplateIdx int,
) (deployment *appsv1.Deployment, continueReconciliation bool, erro error) {
	log := log.FromContext(ctx)
	deployment = &appsv1.Deployment{}
	continueReconciliation = false
	deploymentTemplate := intdepl.Spec.DeploymentTemplates[deployTemplateIdx]

	resourceName := r.getDeploymentFullName(intdepl.Name, deploymentTemplate.Name)
	resourceKey := types.NamespacedName{Name: resourceName, Namespace: intdepl.ObjectMeta.Namespace} 
	if err := r.Get(ctx, resourceKey, deployment); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "failed to fetch deployment")	
			return nil, false, err
		}

		templateCopy := deploymentTemplate.Template.DeepCopy()
		templateCopy.Template.Spec.SchedulerName = INTERNAL_TELEMETRY_SCHEDULER_NAME
		templateCopy.Template.Labels[INTERNAL_TELEMETRY_POD_DEPLOYMENT_OWNER_LABEL] = intdepl.Name
		templateCopy.Template.Labels[INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL] = deploymentTemplate.Name
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: resourceName,
				Namespace: intdepl.Namespace,
			},
			Spec: *templateCopy,
		}

		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "failed to create deployment")
			return nil, false, err
		}
		return deployment, false, nil
	}

	// TODO check if spec matches

	return deployment, true, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InternalInNetworkTelemetryDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *InternalInNetworkTelemetryDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	
	intdepl := &incv1alpha1.InternalInNetworkTelemetryDeployment{}
	if err := r.Get(ctx, req.NamespacedName, intdepl); err != nil {
		log.Error(err, "deleted")
		return ctrl.Result{}, nil
	}

	if len(intdepl.Spec.DeploymentTemplates) != 2 {
		// TODO at validation stage
		err := errors.New("internal INT deployment requires 2 deployment templates")
		log.Error(err, "invalid resource")
		return ctrl.Result{Requeue: false}, nil
	}

	deployments := []*appsv1.Deployment{}
	for i := range intdepl.Spec.DeploymentTemplates {
		deployment, continueReconciliation, err := r.reconcileDeployment(ctx, intdepl, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !continueReconciliation {
			return ctrl.Result{Requeue: true}, nil
		}
		deployments = append(deployments, deployment)
	}

	_ = deployments

	// deployments := &appsv1.DeploymentList{}
	// r.List(ctx, deployments, )

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InternalInNetworkTelemetryDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.InternalInNetworkTelemetryDeployment{}).
		Complete(r)
}
