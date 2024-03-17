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

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	"github.com/Fl0k3n/k8s-inc/inc-operator/controllers/utils"
	"github.com/Fl0k3n/k8s-inc/inc-operator/shimutils"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ExternalInNetworkTelemetryEndpointsReconciler reconciles a ExternalInNetworkTelemetryEndpoints object
type ExternalInNetworkTelemetryEndpointsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const ANY_IPv4 = "any"

//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=services/status,verbs=get;

// func (r *ExternalInNetworkTelemetryEndpointsReconciler) handleEntriesReconciliation(
// 	endpointsResource incv1alpha1.ExternalInNetworkTelemetryEndpoints,
// 	entries []incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry,
// 	sinks DependingEntries[Edge],
// 	sources DependingEntries[Edge],
// 	transits DependingEntries[string],
// ) error {
// 	shimService := shimutils.NewSdnShimService()
// 	failures := map[string]struct{}{}

// 	for switchName, entry := range transits.Entries {
// 		shimService.AssertIsTransit()
// 	}

// 	for i, entry := range entries {
// 		if entry.EntryStatus == incv1alpha1.EP_PENDING {
// 			entries[i].EntryStatus = incv1alpha1.EP_READY
// 		}
// 	}
// }


func (r *ExternalInNetworkTelemetryEndpointsReconciler) withTelemetryClient(grpcAddr string, consumer func(pbt.TelemetryServiceClient) error) error {
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	return consumer(pbt.NewTelemetryServiceClient(conn))
}


func (r *ExternalInNetworkTelemetryEndpointsReconciler) buildIngressEntities(
	ctx context.Context,
	endpoints *incv1alpha1.ExternalInNetworkTelemetryEndpoints,
) ([]*pbt.RawTelemetryEntity, error) {
	ingressInfo := endpoints.Spec.IngressInfo
	if ingressInfo.IngressType != incv1alpha1.NODE_PORT {
		return nil, fmt.Errorf("unsupported ingress type %s", ingressInfo.IngressType)
	}

	nodePortSvc := &v1.Service{}
	if err := r.Get(ctx, client.ObjectKey{
			Name: ingressInfo.NodePortServiceName,
			Namespace: ingressInfo.NodePortServiceNamespace,
		}, nodePortSvc); err != nil {
		return nil, err	
	}

	if nodePortSvc.Spec.Type != v1.ServiceTypeNodePort {
		return nil, fmt.Errorf("unexpected service type")
	}

	if len(nodePortSvc.Spec.Ports) == 0 {
		return nil, fmt.Errorf("nodePort service's ports are unconfigured")
	}

	port := nodePortSvc.Spec.Ports[0].NodePort
	res := []*pbt.RawTelemetryEntity{}
	for _, nodeName := range ingressInfo.NodeNames {
		res = append(res, &pbt.RawTelemetryEntity{
			DeviceName: nodeName,
			Port: &port,
		})
	}

	return res, nil
}


func (r *ExternalInNetworkTelemetryEndpointsReconciler) reconcilePodTelemetry(
	ctx context.Context,
	endpoints *incv1alpha1.ExternalInNetworkTelemetryEndpoints,
	shim *shimv1alpha1.SDNShim,
) (continueReconciliation bool, res ctrl.Result, err error) {
	log := log.FromContext(ctx)
	targetEntities := map[string]*pbt.TunneledEntities{}
	changed := utils.MonitoringPolicyChanged(endpoints.Spec.MonitoringPolicy, endpoints.Status.InternalMonitoringPolicy) ||
			   utils.IngressInfoChanged(endpoints.Spec.IngressInfo, endpoints.Status.InternalIngressInfo)

	// TODO optimize, don't send n queries
	for _, entry := range endpoints.Spec.Entries {
		addToTargets := false
		if entry.EntryStatus == incv1alpha1.EP_READY {
			addToTargets = true
		} else if entry.EntryStatus == incv1alpha1.EP_TERMINATING {
			changed = true
		} else if entry.EntryStatus == incv1alpha1.EP_PENDING {
			changed = true
			addToTargets = true
		}
		if addToTargets {
			podDetails := &v1.Pod{}
			key := client.ObjectKey{Name: entry.PodReference.Name, Namespace: entry.PodReference.Namespace}
			if err := r.Get(ctx, key, podDetails); err != nil {
				log.Error(err, "Failed to fetch pod details")
				return false, ctrl.Result{}, err
			}
			entity := &pbt.TunneledEntity{
				TunneledIp: podDetails.Status.PodIP,
				Port: nil, // any port
			}
			if oldEntities, ok := targetEntities[entry.NodeName]; ok {
				oldEntities.Entities = append(oldEntities.Entities, entity)
			} else {
				targetEntities[entry.NodeName] = &pbt.TunneledEntities{Entities: []*pbt.TunneledEntity{entity}}
			}
		}
	}
	if !changed {
		return true, ctrl.Result{}, nil
	}
	if endpoints.Spec.IngressInfo.IngressType != incv1alpha1.NODE_PORT {
		return false, ctrl.Result{}, fmt.Errorf("unsupported ingress type %s", endpoints.Spec.IngressInfo.IngressType)
	}

	ingressEntities := map[string]*pbt.TunneledEntities{}
	for _, ingressNode := range endpoints.Spec.IngressInfo.NodeNames {
		ingressEntities[ingressNode] = &pbt.TunneledEntities{
			Entities: []*pbt.TunneledEntity{
				{
					TunneledIp: ANY_IPv4,
					Port: nil,
				},
			},
		}
	}

	sendTelemetryRequest := func(mode utils.TelemetryRequestType) (stop bool) {
		sources := targetEntities
		targets := ingressEntities
		if mode == utils.INGRESS_TO_PODS {
			sources = ingressEntities
			targets = targetEntities
		}
		req := &pbt.EnableTelemetryRequest{
			ProgramName: endpoints.Spec.ProgramName,
			CollectionId: utils.BuildCollectionName(endpoints.Name, mode),
			CollectorNodeName: endpoints.Spec.CollectorNodeName,
			CollectorPort: 6000, // TODO
			Sources: &pbt.EnableTelemetryRequest_TunneledSources{
				TunneledSources: &pbt.TunneledTelemetryEntities{
					DeviceNamesWithEntities: sources,	
				},
			},
			Targets: &pbt.EnableTelemetryRequest_TunneledTargets{
				TunneledTargets: &pbt.TunneledTelemetryEntities{
					DeviceNamesWithEntities: targets,	
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


	if endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_ALL ||
	   endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_EXTERNAL_TO_PODS {
		if stop := sendTelemetryRequest(utils.INGRESS_TO_PODS); stop {
			return
		}
	}

	if endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_ALL ||
	   endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_PODS_TO_EXTERNAL {
		if stop := sendTelemetryRequest(utils.PODS_TO_INGRESS); stop {
			return
		}
	}

	for i := range endpoints.Spec.Entries {
		endpoints.Spec.Entries[i].EntryStatus = incv1alpha1.EP_READY
	}
	if err = r.Update(ctx, endpoints); err != nil {
		log.Error(err, "Failed to update endpoints even though requests went through")
		return	
	}
	
	endpoints.Status.InternalIngressInfo = &endpoints.Spec.IngressInfo
	endpoints.Status.InternalMonitoringPolicy = &endpoints.Spec.MonitoringPolicy
	
	err = r.Status().Update(ctx, endpoints)
	continueReconciliation = false
	res = ctrl.Result{Requeue: true}
	return
}

func (r *ExternalInNetworkTelemetryEndpointsReconciler) reconcileIngressTelemetry(
	ctx context.Context,
	endpoints *incv1alpha1.ExternalInNetworkTelemetryEndpoints,
	topo *shimv1alpha1.Topology,
	shim *shimv1alpha1.SDNShim,
) (continueReconciliation bool, res ctrl.Result, err error) {
	continueReconciliation = true
	err = nil
	changed := utils.MonitoringPolicyChanged(endpoints.Spec.MonitoringPolicy, endpoints.Status.ExternalMonitoringPolicy) ||
			   utils.IngressInfoChanged(endpoints.Spec.IngressInfo, endpoints.Status.ExternalIngressInfo)
	if !changed {
		return
	}
	log := log.FromContext(ctx)
	G := shimutils.TopologyToGraph(topo)

	// TODO cleanup old monitoring if needed
	externalDevices := shimutils.GetExternalDevices(G)
	externalDeviceEntities := []*pbt.RawTelemetryEntity{}
	for _, dev := range externalDevices {
		externalDeviceEntities = append(externalDeviceEntities, &pbt.RawTelemetryEntity{
			DeviceName: dev.Name,
			Port: nil,
		})
	}

	ingressDeviceEntities, er := r.buildIngressEntities(ctx, endpoints)
	if er != nil {
		log.Error(er, "Failed to build ingress entities")
		err = er
		return
	}

	sendTelemetryRequest := func(mode utils.TelemetryRequestType) (stop bool) {
		sources := externalDeviceEntities
		targets := ingressDeviceEntities
		if mode == utils.INGRESS_TO_EXTERNAL {
			sources = ingressDeviceEntities
			targets = externalDeviceEntities
		}
		req := &pbt.EnableTelemetryRequest{
			ProgramName: endpoints.Spec.ProgramName,
			CollectionId: utils.BuildCollectionName(endpoints.Name, mode),
			CollectorNodeName: endpoints.Spec.CollectorNodeName,
			CollectorPort: 6000, // TODO
			Sources: &pbt.EnableTelemetryRequest_RawSources{
				RawSources: &pbt.RawTelemetryEntities{
					Entities: sources,
				},
			},
			Targets: &pbt.EnableTelemetryRequest_RawTargets{
				RawTargets: &pbt.RawTelemetryEntities{
					Entities: targets,
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

	if endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_ALL ||
	   endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_EXTERNAL_TO_PODS {
		if stop := sendTelemetryRequest(utils.EXTERNAL_TO_INGRESS); stop {
			return
		}
	}

	if endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_ALL ||
	   endpoints.Spec.MonitoringPolicy == incv1alpha1.MONITOR_PODS_TO_EXTERNAL {
		if stop := sendTelemetryRequest(utils.INGRESS_TO_EXTERNAL); stop {
			return
		}
	}
	
	endpoints.Status.ExternalIngressInfo = &endpoints.Spec.IngressInfo
	endpoints.Status.ExternalMonitoringPolicy = &endpoints.Spec.MonitoringPolicy
	
	err = r.Status().Update(ctx, endpoints)
	continueReconciliation = false
	res = ctrl.Result{Requeue: true}
	return
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ExternalInNetworkTelemetryEndpoints object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ExternalInNetworkTelemetryEndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Handling endpoints")

	endpoints := &incv1alpha1.ExternalInNetworkTelemetryEndpoints{}
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
	topo, err := shimutils.LoadTopology(ctx, r.Client)
	if err != nil {
		log.Error(err, "Failed to load topology")
		return ctrl.Result{}, err
	}

	if continueReconciliation, res, err := r.reconcileIngressTelemetry(ctx, endpoints, topo, shim); !continueReconciliation {
		return res, err	
	}

	if continueReconciliation, res, err := r.reconcilePodTelemetry(ctx, endpoints, shim); !continueReconciliation {
		return res, err	
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExternalInNetworkTelemetryEndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.ExternalInNetworkTelemetryEndpoints{}).
		Complete(r)
}
