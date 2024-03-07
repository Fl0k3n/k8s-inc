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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"

	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
)

// SDNShimReconciler reconciles a SDNShim object
type SDNShimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	sdnClient pb.SdnFrontendClient
}

func (r *SDNShimReconciler) runSdnClient(grpcAddr string) error {
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	r.sdnClient = pb.NewSdnFrontendClient(conn)
	return nil
}

func (r *SDNShimReconciler) buildClusterTopoCR(topo *pb.TopologyResponse, req ctrl.Request) *incv1alpha1.Topology {
	graph := make([]incv1alpha1.NetworkDevice, len(topo.Graph))
	for i, dev := range topo.Graph {
		links := make([]incv1alpha1.Link, len(dev.Links))
		for j, link := range dev.Links {
			links[j] = incv1alpha1.Link{
				PeerName: link.PeerName,
			}
		}
		graph[i] = incv1alpha1.NetworkDevice{
			Name: dev.Name,
			DeviceType: dev.DeviceType.String(),
			Links: links,
		}
	}
	
	res := &incv1alpha1.Topology{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-topo",
			Namespace: req.Namespace,
		},
		Spec: incv1alpha1.TopologySpec{
			Graph: graph,
		},
		Status: incv1alpha1.TopologyStatus{
			CustomStatus: "Creating",
		},
	}
	return res
}

func (r *SDNShimReconciler) reconcileTopology(ctx context.Context, req ctrl.Request) (changed bool, err error) {
	topos := &incv1alpha1.TopologyList{}
	if err := r.List(ctx, topos); err != nil {
		return false, err
	}
	if topos.Items == nil || len(topos.Items) == 0 {
		topo, err := r.sdnClient.GetTopology(ctx, &emptypb.Empty{})
		if err != nil {
			return false, err
		}
		topoCr := r.buildClusterTopoCR(topo, req)
		if err := r.Client.Create(ctx, topoCr); err != nil {
			log.FromContext(ctx).Error(err, "Failed to create CR")
			return false, err
		}
		return true, nil
	} 
	log.FromContext(ctx).Info("Topo exists, TODO reconcile it if it may change") // TODO
	return false, nil
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SDNShim object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *SDNShimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	
	shimConfig := &incv1alpha1.SDNShim{}
	err := r.Get(ctx, req.NamespacedName, shimConfig)
	if err != nil {
		log.Info("SdnShim not found")
		return ctrl.Result{}, nil
	}

	if shimConfig.Spec.SdnConfig.SdnType == "microsdn" {
		if r.sdnClient == nil {
			if err := r.runSdnClient(shimConfig.Spec.SdnConfig.SdnGrpcAddr); err != nil {
				log.Error(err, "Failed to connect to SDN controller")
				return ctrl.Result{}, err
			}
		}
		changed, err := r.reconcileTopology(ctx, req)
		if err != nil {
			return ctrl.Result{}, err
		}
		if changed {
			log.Info("Updated topology")
		}
	} else {
		return ctrl.Result{}, errors.New("unsupported SDN type")
	}
	

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SDNShimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.SDNShim{}).
		Complete(r)
}
