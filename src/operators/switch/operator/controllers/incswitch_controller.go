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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	incv1alpha1 "github.com/example/memcached-operator/api/v1alpha1"
)

const (
	typeAvailableIncSwitch = "Available"
	typeDegradedIncSwitch = "Degraded"
)

// IncSwitchReconciler reconciles a IncSwitch object
type IncSwitchReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func buildGraph(switches incv1alpha1.IncSwitchList) string {
	graph := ""
	indent := "    "
	for _, s := range switches.Items {
		graph += fmt.Sprintf("%s: \n", s.Name)
		if s.Spec.Links == nil || len(s.Spec.Links) == 0 {
			graph += fmt.Sprintf("%sempty", indent)
		} else {
			for _, link := range s.Spec.Links {
				graph += fmt.Sprintf("%s->%s\n", indent, link.PeerName)
			}
		}
		graph += "\n"
	}	
	return graph
}

//+kubebuilder:rbac:groups=inc.example.com,resources=incswitches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.example.com,resources=incswitches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.example.com,resources=incswitches/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IncSwitch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *IncSwitchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	incSwitch := &incv1alpha1.IncSwitch{}
	err := r.Get(ctx, req.NamespacedName, incSwitch)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("IncSwitch resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	} else {
		fmt.Println(incSwitch)
	}

	allIncSwitches := incv1alpha1.IncSwitchList{}
	err = r.List(ctx, &allIncSwitches, client.InNamespace(req.Namespace))
	if err != nil {
		log.Error(err, "failed to fetch all switches")
	} else {
		log.Info(fmt.Sprintf("graph: \n%s", buildGraph(allIncSwitches)))
	}

	if incSwitch.Status.Conditions == nil || len(incSwitch.Status.Conditions) == 0 {
		meta.SetStatusCondition(&incSwitch.Status.Conditions, metav1.Condition{
			Type: typeAvailableIncSwitch, 
			Status: metav1.ConditionUnknown,
			Reason: "Reconciling",
			Message: "Starting reconciliation",
		})
		incSwitch.Status.MyOwnStatusField = "test my own field"

		if err = r.Status().Update(ctx, incSwitch); err != nil {
			log.Error(err, "Failed to update incSwitch status")
			return ctrl.Result{}, err
		}

		if err := r.Get(ctx, req.NamespacedName, incSwitch); err != nil {
			log.Error(err, "Failed to re-fetch incsswitch")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IncSwitchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.IncSwitch{}).
		Complete(r)
}
