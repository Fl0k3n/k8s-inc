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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const INTERNAL_TELEMETRY_SCHEDULER_NAME = "internal-telemetry"
const INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL = "inc.kntp.com/owned-by-iintdepl"
const INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL = "inc.kntp.com/part-of-deployment"

const INTERNAL_TELEMETRY_ENDPOINTS_FINALIZER = "inc.kntp.com/iint-finalizer"

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


func (r *InternalInNetworkTelemetryDeploymentReconciler) reconcileDeploymentEndpointEntries(
	ctx context.Context,
	pods []v1.Pod, 
	endpointEntries []incv1alpha1.InternalInNetworkTelemetryEndpointsEntry,
) (_ []incv1alpha1.InternalInNetworkTelemetryEndpointsEntry, changed bool) {
	log := log.FromContext(ctx)	
	changed = false
	validEntries := []incv1alpha1.InternalInNetworkTelemetryEndpointsEntry{}
	// check if for each managed endpoint pod exists, if not delete it
	for i, ep := range endpointEntries {
		found := false
		for _, pod := range pods {
			if pod.UID == ep.PodReference.UID {
				found = true
				break
			}
		}
		if !found {
			if ep.EntryStatus != incv1alpha1.EP_TERMINATING {
				changed = true
				endpointEntries[i].EntryStatus = incv1alpha1.EP_TERMINATING
			}
			validEntries = append(validEntries, endpointEntries[i])
		}
	}

	// check if for each managed pod there exists managed endpoint, if not create new
	for _, pod := range pods {
		var ep *incv1alpha1.InternalInNetworkTelemetryEndpointsEntry = nil
		for i, e := range endpointEntries {
			if e.PodReference.UID == pod.UID {
				ep = &endpointEntries[i]
				break
			}
		}
		if ep == nil {
			// TODO maybe it can be done before pod is running?
			if pod.Status.Phase == v1.PodRunning {
				// create managed endpoint, defer INC actions to its controller
				changed = true
				validEntries = append(validEntries, incv1alpha1.InternalInNetworkTelemetryEndpointsEntry{
					PodReference: v1.ObjectReference{
						Kind: pod.Kind,
						Namespace: pod.Namespace,
						Name: pod.Name,
						UID: pod.UID,
						APIVersion: pod.APIVersion,
					},
					EntryStatus: incv1alpha1.EP_PENDING,
					NodeName: pod.Spec.NodeName,
					PodIp: pod.Status.PodIP,
				})
			}
		} else {
			// check if its still relevant and update
			log.Info("Checking state of pod-ep...")
			validEntries = append(validEntries, *ep)
		}
	}
	return validEntries, changed
}

func (r *InternalInNetworkTelemetryDeploymentReconciler) reconcileDeploymentEndpoints(
	ctx context.Context,
	perDeploymentPods map[string][]v1.Pod,
	intdepl *incv1alpha1.InternalInNetworkTelemetryDeployment,
	managedEndpoints *incv1alpha1.InternalInNetworkTelemetryEndpoints,
) (ctrl.Result, bool, error) {
	log := log.FromContext(ctx)	

	reconciledEntries := make([]incv1alpha1.DeploymentEndpoints, len(managedEndpoints.Spec.DeploymentEndpoints))
	changed := false
	for _, depl := range intdepl.Spec.DeploymentTemplates {
		endpointsIdx := -1
		for i, entry := range managedEndpoints.Spec.DeploymentEndpoints {
			if entry.DeploymentName == depl.Name {
				endpointsIdx = i
				break
			}
		}
		if endpointsIdx == -1 {
			return ctrl.Result{}, false, fmt.Errorf("invalid endpoints state, no deployment %s", depl.Name)
		}
		pods := perDeploymentPods[depl.Name]
		endpoints := managedEndpoints.Spec.DeploymentEndpoints[endpointsIdx]
		entries, chngd := r.reconcileDeploymentEndpointEntries(ctx, pods, endpoints.Entries)
		changed = changed || chngd
		reconciledEntries[endpointsIdx] = incv1alpha1.DeploymentEndpoints{
			DeploymentName: endpoints.DeploymentName,
			Entries: entries,
		}
	}

	if changed {
		managedEndpoints.Spec.DeploymentEndpoints = reconciledEntries
		log.Info("Updating managed endpoints")
		if err := r.Update(ctx, managedEndpoints); err != nil {
			log.Error(err, "Failed to update managed endpoints")
			return ctrl.Result{}, false, client.IgnoreNotFound(err)
		}
	}

	return ctrl.Result{}, true, nil
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

	isMarkedForDeletion := intdepl.GetDeletionTimestamp() != nil
	if isMarkedForDeletion {
		// there should be finalizers for endpoints, when endpoints are deleted these finalizers should
		// also be removed automatically, then this one will be deleted, no additional action neccessary
		// TODO: set some status 
		return ctrl.Result{}, nil
	}

	if len(intdepl.Spec.DeploymentTemplates) != 2 {
		// TODO at validation stage
		err := errors.New("internal INT deployment requires 2 deployment templates")
		log.Error(err, "invalid resource")
		return ctrl.Result{Requeue: false}, nil
	}

	deployments := map[string]*appsv1.Deployment{}
	someCreated := false
	for i := range intdepl.Spec.DeploymentTemplates {
		deploymentTemplate := intdepl.Spec.DeploymentTemplates[i]
		resourceKey := resourceKeyForIntdepl(intdepl, deploymentTemplate.Name)
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, resourceKey, deployment); err != nil {
			if apierrors.IsNotFound(err) {
				deployment = createDeploymentForIntdepl(intdepl, deploymentTemplate, resourceKey)
				if err := ctrl.SetControllerReference(intdepl, deployment, r.Scheme); err != nil {
					log.Error(err, "failed to set controller reference to deployment")
					return ctrl.Result{}, err
				}
				if err := r.Create(ctx, deployment); err != nil {
					log.Error(err, "failed to create deployment")
					return ctrl.Result{}, err
				}
				someCreated = true
			} else {
				log.Error(err, "failed to fetch deployment")
				return ctrl.Result{}, err
			}
		} else {
			// TODO: check if deployment wasn't changed
		}
		deployments[deploymentTemplate.Name] = deployment
	}
	if someCreated {
		return ctrl.Result{Requeue: true}, nil
	}
	
	managedEndpoints := &incv1alpha1.InternalInNetworkTelemetryEndpoints{}
	managedEndpointsKey := client.ObjectKey{Name: req.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, managedEndpointsKey, managedEndpoints); err != nil {
		if apierrors.IsNotFound(err) {
			entries := []incv1alpha1.DeploymentEndpoints{}
			for _, deplTemplate := range intdepl.Spec.DeploymentTemplates {
				entries = append(entries, incv1alpha1.DeploymentEndpoints{	
					DeploymentName: deplTemplate.Name,
					Entries: []incv1alpha1.InternalInNetworkTelemetryEndpointsEntry{},
				})
			}
			eps := &incv1alpha1.InternalInNetworkTelemetryEndpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: managedEndpointsKey.Name,
					Namespace: managedEndpointsKey.Namespace,
				},
				Spec: incv1alpha1.InternalInNetworkTelemetryEndpointsSpec{
					DeploymentEndpoints: entries,
					CollectorRef: intdepl.Spec.CollectorRef,
				},
			}
			if err := controllerutil.SetControllerReference(intdepl, eps, r.Scheme); err != nil {
				log.Error(err, "failed to set controller reference")
				return ctrl.Result{}, err
			}
			controllerutil.AddFinalizer(eps, INTERNAL_TELEMETRY_ENDPOINTS_FINALIZER)
			if err := r.Create(ctx, eps); err != nil {
				log.Error(err, "failed to create endpoints")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} 
		return ctrl.Result{}, err
	}

	perDeploymentPods, err := r.loadPerDeploymentPods(ctx, deployments)
	if err != nil {
		log.Error(err, "Failed to load pods")
		return ctrl.Result{}, err
	}

	if res, continueReconciliation, err := r.reconcileDeploymentEndpoints(ctx, perDeploymentPods, intdepl, managedEndpoints); !continueReconciliation {
		return res, err
	}

	return ctrl.Result{}, nil
}

func (r *InternalInNetworkTelemetryDeploymentReconciler) loadPerDeploymentPods(
	ctx context.Context,
	deployments map[string]*appsv1.Deployment,
) (map[string][]v1.Pod, error) {
	perDeploymentPods := map[string][]v1.Pod{}
	for _, depl := range deployments {
		podSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: depl.Spec.Template.Labels,
		})
		if err != nil {
			return nil, err
		}
		pods := &v1.PodList{}
		listOptions := &client.ListOptions{
			LabelSelector: podSelector,
			Namespace: depl.Namespace,
		}
		if err := r.List(ctx, pods, listOptions); err != nil {
			return nil, err
		}
		perDeploymentPods[depl.Name] = pods.Items
	}
	return perDeploymentPods, nil
}

func createDeploymentForIntdepl(
	intdepl *incv1alpha1.InternalInNetworkTelemetryDeployment,
	depl incv1alpha1.NamedDeploymentSpec,
	resourceKey types.NamespacedName,
) *appsv1.Deployment {
	depl.Template.Template.Spec.SchedulerName = INTERNAL_TELEMETRY_SCHEDULER_NAME
	labels := labelsForDeploymentPods(intdepl, depl.Name)
	depl.Template.Template.Labels = labels
	depl.Template.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceKey.Name,
			Namespace: resourceKey.Namespace,
		},
		Spec: depl.Template,
	}
}

func labelsForDeploymentPods(intdepl *incv1alpha1.InternalInNetworkTelemetryDeployment, deplName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name": "internal-INT-pod",
		"app.kubernetes.io/instance":   intdepl.Name,
		"app.kubernetes.io/created-by": "controller-manager",
		INTERNAL_TELEMETRY_POD_INTDEPL_NAME_LABEL: intdepl.Name,
		INTERNAL_TELEMETRY_POD_DEPLOYMENT_NAME_LABEL: deplName,
	}
}

func resourceKeyForIntdepl(intdepl *incv1alpha1.InternalInNetworkTelemetryDeployment, deplName string) types.NamespacedName {
	return types.NamespacedName{
		Name: fmt.Sprintf("%s-%s", intdepl.Name, deplName),
		Namespace: intdepl.Namespace,
	} 
}

// SetupWithManager sets up the controller with the Manager.
func (r *InternalInNetworkTelemetryDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.InternalInNetworkTelemetryDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&incv1alpha1.ExternalInNetworkTelemetryEndpoints{}).
		Complete(r)
}
