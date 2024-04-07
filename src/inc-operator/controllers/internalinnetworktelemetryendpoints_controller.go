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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	"github.com/Fl0k3n/k8s-inc/inc-operator/controllers/utils"
	"github.com/Fl0k3n/k8s-inc/inc-operator/shimutils"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
)

// InternalInNetworkTelemetryEndpointsReconciler reconciles a InternalInNetworkTelemetryEndpoints object
type InternalInNetworkTelemetryEndpointsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetryendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetryendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetryendpoints/finalizers,verbs=update


func (r *InternalInNetworkTelemetryEndpointsReconciler) withTelemetryClient(grpcAddr string, consumer func(pbt.TelemetryServiceClient) error) error {
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	return consumer(pbt.NewTelemetryServiceClient(conn))
}

func (r *InternalInNetworkTelemetryEndpointsReconciler) reconcileDeploymentEndpoints(
	deplEndpoints incv1alpha1.DeploymentEndpoints,
) (res map[string]*pbt.TunneledEntities, changed bool) {
	res = make(map[string]*pbt.TunneledEntities)
	changed = false
	for _, entry := range deplEndpoints.Entries {
		valid := false
		if entry.EntryStatus == incv1alpha1.EP_READY {
			valid = true
		} else if entry.EntryStatus == incv1alpha1.EP_TERMINATING {
			changed = true
		} else if entry.EntryStatus == incv1alpha1.EP_PENDING {
			changed = true
			valid = true
		}
		if valid {
			entity := &pbt.TunneledEntity{
				TunneledIp: entry.PodIp,
				Port: nil, // any port
			}
			if oldEntities, ok := res[entry.NodeName]; ok {
				oldEntities.Entities = append(oldEntities.Entities, entity)
			} else {
				res[entry.NodeName] = &pbt.TunneledEntities{Entities: []*pbt.TunneledEntity{entity}}
			}
		}
	}
	return
}

func (r *InternalInNetworkTelemetryEndpointsReconciler) reconcilePodTelemetry(
	ctx context.Context,
	endpoints *incv1alpha1.InternalInNetworkTelemetryEndpoints,
	shim *shimv1alpha1.SDNShim,
) (continueReconciliation bool, res ctrl.Result, err error) {
	log := log.FromContext(ctx)
	continueReconciliation = false
	perDeplEntities := map[string]map[string]*pbt.TunneledEntities{}
	changed := false
	for _, deplEndpoints := range endpoints.Spec.DeploymentEndpoints {
		entities, chngd := r.reconcileDeploymentEndpoints(deplEndpoints)
		changed = changed || chngd
		perDeplEntities[deplEndpoints.DeploymentName] = entities
	}
	if !changed {
		return true, ctrl.Result{}, nil
	}
	sendTelemetryRequest := func(sourceDepl string, targetDepl string) (stop bool) {
		req := &pbt.EnableTelemetryRequest{
			CollectionId: utils.BuildInternalCollectionName(endpoints.Name, sourceDepl, targetDepl),
			CollectorNodeName: endpoints.Spec.CollectorNodeName,
			CollectorPort: 6000, // TODO
			Sources: &pbt.EnableTelemetryRequest_TunneledSources{
				TunneledSources: &pbt.TunneledTelemetryEntities{
					DeviceNamesWithEntities: perDeplEntities[sourceDepl],	
				},
			},
			Targets: &pbt.EnableTelemetryRequest_TunneledTargets{
				TunneledTargets: &pbt.TunneledTelemetryEntities{
					DeviceNamesWithEntities: perDeplEntities[targetDepl],	
				},
			},
		}
		var resp *pbt.EnableTelemetryResponse
		err = r.withTelemetryClient(shim.Spec.SdnConfig.SdnGrpcAddr, func(tsc pbt.TelemetryServiceClient) error {
			response, er := tsc.EnableTelemetry(ctx, req)
			if er != nil {
				return er
			}
			resp = response
			return nil
		})
		if err != nil {
			return true
		}
		if resp.TelemetryState == pbt.TelemetryState_IN_PROGRESS || resp.TelemetryState == pbt.TelemetryState_FAILED {
			continueReconciliation = false
			res = ctrl.Result{RequeueAfter: time.Second * 5}
			return true
		}
		return false
	}
	eps := endpoints.Spec.DeploymentEndpoints
	if len(eps) != 2 {
		return false, ctrl.Result{}, fmt.Errorf("2 deployments are required")
	}
	if stop := sendTelemetryRequest(eps[0].DeploymentName, eps[1].DeploymentName); stop {
		return
	}
	if stop := sendTelemetryRequest(eps[1].DeploymentName, eps[0].DeploymentName); stop {
		return
	}
	for i, eps := range endpoints.Spec.DeploymentEndpoints {
		for j := range eps.Entries {
			endpoints.Spec.DeploymentEndpoints[i].Entries[j].EntryStatus = incv1alpha1.EP_READY
		}
	}
	if err = r.Update(ctx, endpoints); err != nil {
		log.Error(err, "Failed to update endpoints even though requests went through")
		return	
	}
	continueReconciliation = false
	res = ctrl.Result{Requeue: true}
	return
}


// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InternalInNetworkTelemetryEndpoints object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *InternalInNetworkTelemetryEndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling internal endpoints")

	endpoints := &incv1alpha1.InternalInNetworkTelemetryEndpoints{}
	if err := r.Get(ctx, req.NamespacedName, endpoints); err != nil {
		// finalizers here or in depl?
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	shim, err := shimutils.LoadSDNShim(ctx, r.Client)
	if err != nil {
		log.Error(err, "Failed to load SDNShim config")
		return ctrl.Result{}, err
	}

	if continueReconciliation, res, err := r.reconcilePodTelemetry(ctx, endpoints, shim); !continueReconciliation {
		return res, err	
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InternalInNetworkTelemetryEndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.InternalInNetworkTelemetryEndpoints{}).
		Complete(r)
}
