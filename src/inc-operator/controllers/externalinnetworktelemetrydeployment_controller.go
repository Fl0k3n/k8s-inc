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

	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	"github.com/Fl0k3n/k8s-inc/inc-operator/shimutils"
)

// ExternalInNetworkTelemetryDeploymentReconciler reconciles a ExternalInNetworkTelemetryDeployment object
type ExternalInNetworkTelemetryDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetrydeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetrydeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetrydeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=deployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=pods/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=replicasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=replicasets/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/finalizers,verbs=update


const POD_EINT_DEPL_OWNER_LABEL = "inc.kntp.com/deployed-by"

func (r *ExternalInNetworkTelemetryDeploymentReconciler) pickFeasibleNodes(
		ctx context.Context,
		topo *shimv1alpha1.Topology,
		req ctrl.Request,
		programName string) ([]string, error) {
	log := log.FromContext(ctx)
	_ = topo
	_ = req
	incSwitches := &shimv1alpha1.IncSwitchList{}
	if err := r.List(ctx, incSwitches); err != nil {
		log.Error(err, "Failed to load inc switches")
		return nil, err
	}

	// TODO proper selection logic
	hasOneWithProgram := false
	for _, incSwitch := range incSwitches.Items {
		// if incSwitch.Status.InstalledProgram == programName { TODO
		if incSwitch.Spec.ProgramName == programName {
			hasOneWithProgram = true
			break
		}
	}
	if !hasOneWithProgram {
		return []string{}, nil
	}

	return []string{"w1"}, nil
}


func (r *ExternalInNetworkTelemetryDeploymentReconciler) reconcileEndpoints(
		ctx context.Context,
		managedPods *v1.PodList,
		managedEndpoints *incv1alpha1.ExternalInNetworkTelemetryEndpoints) (ctrl.Result, error) {
	log := log.FromContext(ctx)	
	changed := false
	validEntries := []incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry{}
	
	// check if for each managed endpoint pod exists, if not delete it
	for _, ep := range managedEndpoints.Spec.Entries {
		found := false
		for _, pod := range managedPods.Items {
			if pod.UID == ep.PodReference.UID {
				found = true
				break
			}
		}
		if !found {
			if ep.EntryStatus != incv1alpha1.EP_TERMINATING {
				changed = true
				ep.EntryStatus = incv1alpha1.EP_TERMINATING
			}
			validEntries = append(validEntries, ep)
		}
	}

	// check if for each managed pod there exists managed endpoint, if not create new
	for _, pod := range managedPods.Items {
		var ep *incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry = nil
		for _, m := range managedEndpoints.Spec.Entries {
			if m.PodReference.UID == pod.UID {
				ep = &m
				break
			}
		}
		if ep == nil {
			// TODO maybe it can be done before pod is running?
			if pod.Status.Phase == v1.PodRunning {
				// create managed endpoint, defer INC actions to its controller
				changed = true
				validEntries = append(validEntries, incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry{
					PodReference: v1.ObjectReference{
						Kind: pod.Kind,
						Namespace: pod.Namespace,
						Name: pod.Name,
						UID: pod.UID,
						APIVersion: pod.APIVersion,
					},
					EntryStatus: incv1alpha1.EP_PENDING,
					NodeName: pod.Spec.NodeName,
				})
			}
		} else {
			// check if its still relevant and update
			log.Info("Checking state of pod-ep...")
			validEntries = append(validEntries, *ep)
		}
	}

	if changed {
		managedEndpoints.Spec.Entries = validEntries
		log.Info("Updating managed endpoints")
		if err := r.Update(ctx, managedEndpoints); err != nil {
			log.Error(err, "Failed to update managed endpoints")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	return ctrl.Result{}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ExternalInNetworkTelemetryDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ExternalInNetworkTelemetryDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
 	log := log.FromContext(ctx)

	eintDepl := &incv1alpha1.ExternalInNetworkTelemetryDeployment{}
	if err := r.Get(ctx, req.NamespacedName, eintDepl); err != nil {
		// TODO finalizers
		return ctrl.Result{}, nil
	}

	topo, err := shimutils.LoadTopology(ctx, r.Client)
	if err != nil {
		log.Error(err, "Failed to get topology")
		return ctrl.Result{}, err
	}
	
	deploy := &appsv1.Deployment{}

	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: eintDepl.Name}, deploy); err != nil {
		if apierrors.IsNotFound(err) {

			feasibleNodes, err := r.pickFeasibleNodes(ctx, topo, req, eintDepl.Spec.RequiredProgram)
			if err != nil {
				return ctrl.Result{}, err
			}

			if len(feasibleNodes) == 0 {
				log.Info("No feasible nodes")
				return ctrl.Result{}, nil
			}

			templateCopy := eintDepl.Spec.DeploymentTemplate.DeepCopy()
			templateCopy.Template.Labels[POD_EINT_DEPL_OWNER_LABEL] = eintDepl.Name
			templateCopy.Template.Spec.Affinity = &v1.Affinity{
				NodeAffinity: &v1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
						NodeSelectorTerms: []v1.NodeSelectorTerm{
							{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key: "sname",
										Operator: "In",
										Values:	feasibleNodes,
									},
								},
							},
						},
					},
				},
			}

			deploy = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: eintDepl.Name,
					Namespace: req.Namespace,
				},
				Spec: *templateCopy,
			}

			if err := controllerutil.SetControllerReference(eintDepl, deploy, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}

			if err := r.Client.Create(ctx, deploy); err != nil {
				log.Error(err, "Failed to create deployment")
				return ctrl.Result{}, err
			} else {
				log.Info("Created deployment")
				eintDepl.Status.BasicStatus = "deployment created"
				r.Client.Status().Update(ctx, eintDepl)
				return ctrl.Result{Requeue: true}, nil
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	// TODO assert deployment was created by controller 
	managedEndpoints := &incv1alpha1.ExternalInNetworkTelemetryEndpoints{}
	managedEndpointsKey := client.ObjectKey{Name: req.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, managedEndpointsKey, managedEndpoints); err != nil {
		if apierrors.IsNotFound(err) {
			eps := &incv1alpha1.ExternalInNetworkTelemetryEndpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: managedEndpointsKey.Name,
					Namespace: managedEndpointsKey.Namespace,
				},
				Spec: incv1alpha1.ExternalInNetworkTelemetryEndpointsSpec{
					Entries: []incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry{},
					CollectorIpv4: "10.1.1.1", // TODO
				},
			}
			if err := controllerutil.SetControllerReference(eintDepl, eps, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, eps); err != nil {
				log.Error(err, "Failed to create endpoints")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} 
		return ctrl.Result{}, err
	}

	podSelector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}

	// this is after scheduling phase, at this point we must have assured that if there are 
	// any pods they must have been initially placed at valid positions
	// now we need to assure that: 
	// - INC resources are ready for those pods (if pods are running) - we delegate this to endpoints controller
	managedPods := &v1.PodList{}
	listOptions := &client.ListOptions{
		LabelSelector: podSelector,
		Namespace: req.Namespace,
	}
	if err := r.List(ctx, managedPods, listOptions); err != nil {
		log.Error(err, "Failed to load pods")
	}

	return r.reconcileEndpoints(ctx, managedPods, managedEndpoints)
}

func (r *ExternalInNetworkTelemetryDeploymentReconciler) findDeploymentOfManagedPods(podRawObj client.Object) []reconcile.Request {
	// if pods with selector of some eintDepl are changed, trigger reconciliation
	// not all changes are of interest to us (creation, deletion, scheduling, etc), but for now trigger on all
	// finding eintDepl based on labels of a pod IS fragile but leave it for now for simplicity
	pod := podRawObj.(*v1.Pod)
	if pod.ObjectMeta.Labels == nil {
		return []reconcile.Request{}	
	}
	owner, ok := pod.ObjectMeta.Labels[POD_EINT_DEPL_OWNER_LABEL]
	if !ok {
		return []reconcile.Request{}	
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: owner, Namespace: pod.Namespace}}}
}


// SetupWithManager sets up the controller with the Manager.
func (r *ExternalInNetworkTelemetryDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.ExternalInNetworkTelemetryDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&incv1alpha1.ExternalInNetworkTelemetryEndpoints{}). // should we?
		Watches(
			&source.Kind{Type: &v1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(r.findDeploymentOfManagedPods),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}), // TODO more specific
		).
		Complete(r)
}
