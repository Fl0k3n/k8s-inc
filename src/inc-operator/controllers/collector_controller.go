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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
)

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=collectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=collectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=collectors/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=daemonsets/status,verbs=get
//+kubebuilder:rbac:groups=inc.kntp.com,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=services/status,verbs=get
//+kubebuilder:rbac:groups=inc.kntp.com,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=pods/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Collector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *CollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	collector := &incv1alpha1.Collector{}
	if err := r.Get(ctx, req.NamespacedName, collector); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} 
		log.Error(err, "Failed to get collector")
		return ctrl.Result{}, err
	}

	if collector.Status.Conditions == nil || len(collector.Status.Conditions) == 0 {
		meta.SetStatusCondition(&collector.Status.Conditions,  metav1.Condition{
			Type: incv1alpha1.TypeAvailableCollector,
			Status: metav1.ConditionUnknown,
			Reason: "Reconciling",
			Message: "Starting reconciliation",
		})
		if err := r.Status().Update(ctx, collector); err != nil {
			log.Error(err, "Failed to update Collector status")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, collector); err != nil {
			log.Error(err, "Failed to re-fetch collector")
			return ctrl.Result{}, err
		}
	}

	daemonSet := &appsv1.DaemonSet{}
	if err := r.Get(ctx, req.NamespacedName, daemonSet); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Daemonset for collector %s not found, creating", collector.Name))
			daemonSet = r.createDaemonSetForCollector(collector)
			if err := ctrl.SetControllerReference(collector, daemonSet, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference to daemonset")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, daemonSet); err != nil {
				log.Error(err, "Failed to create daemonset")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get daemonset")
		return ctrl.Result{}, err
	}

	collectorService := &v1.Service{}
	if err := r.Get(ctx, req.NamespacedName, collectorService); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Service for collector %s not found, creating", collector.Name))
			collectorService = r.createNodePortServiceForCollector(collector)
			if err := ctrl.SetControllerReference(collector, collectorService, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference to service")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, collectorService); err != nil {
				log.Error(err, "Failed to create service")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get service")
		return ctrl.Result{}, err
	}

	collector.Status.NodeRef = nil
	if daemonSet.Status.DesiredNumberScheduled > 0 {
		podSelector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: labelsForCollector(collector),
		})
		listOptions := &client.ListOptions{
			LabelSelector: podSelector,
			Namespace: req.Namespace,
		}
		pods := &v1.PodList{}
		if err := r.List(ctx, pods, listOptions); err != nil {
			log.Error(err, "Failed to load daemonset pods")
			return ctrl.Result{}, err
		}
		for _, p := range pods.Items {
			if p.Status.Phase == v1.PodRunning {
				collector.Status.NodeRef = &v1.LocalObjectReference{Name: p.Spec.NodeName}
				break
			}
		}
	}

	collector.Status.Port = nil
	// if collectorService.Status.Conditions
	if len(collectorService.Spec.Ports) > 0 {
		nodePort := collectorService.Spec.Ports[0].NodePort
		if nodePort != 0 {
			collector.Status.Port = &nodePort
		} 
	}

	if collector.Status.NodeRef != nil && collector.Status.Port != nil {
		meta.SetStatusCondition(&collector.Status.Conditions, metav1.Condition{
			Type: incv1alpha1.TypeAvailableCollector,
			Status: metav1.ConditionTrue,
			Reason: "Reconciling",
			Message: "Daemonset pod and NodePort are ready",
		})
	} else {
		meta.SetStatusCondition(&collector.Status.Conditions, metav1.Condition{
			Type: incv1alpha1.TypeAvailableCollector,
			Status: metav1.ConditionFalse,
			Reason: "Reconciling",
			Message: "Daemonset pod or node port not ready",
		})
	}

	if err := r.Status().Update(ctx, collector); err != nil {
		log.Error(err, "Failed to update Collector status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CollectorReconciler) createDaemonSetForCollector(collector *incv1alpha1.Collector) *appsv1.DaemonSet {
	labels := labelsForCollector(collector)
	template := collector.Spec.PodSpec

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: collector.Name,
			Namespace: collector.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: template,
			},	
		},
	}
}

func (r *CollectorReconciler) createNodePortServiceForCollector(collector *incv1alpha1.Collector) *v1.Service {
	labels := labelsForCollector(collector)	
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: collector.Name,
			Namespace: collector.Namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: labels,
			Type: v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				{
					Protocol: v1.ProtocolUDP,
					Port: collector.Spec.PodSpec.Containers[0].Ports[0].ContainerPort,
				},
			},
		},
	}
}

func labelsForCollector(collector *incv1alpha1.Collector) map[string]string {
	var imageTag string
	image := collector.Spec.PodSpec.Containers[0].Image
	imageAndTag := strings.Split(image, ":")
	imageTag = "0.0.0"
	if len(imageAndTag) > 1 {
		imageTag = imageAndTag[1]
	}
	return map[string]string{
		"app.kubernetes.io/name": "int-collector",
		"app.kubernetes.io/instance":   collector.Name,
		"app.kubernetes.io/version":    imageTag,
		"app.kubernetes.io/part-of":    "project",
		"app.kubernetes.io/created-by": "controller-manager",
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.Collector{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&v1.Service{}).
		Complete(r)
}
