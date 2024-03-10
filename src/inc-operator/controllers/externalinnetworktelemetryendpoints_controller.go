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
	"time"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	"github.com/Fl0k3n/k8s-inc/inc-operator/shimutils"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=externalinnetworktelemetryendpoints/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/finalizers,verbs=update


type Edge struct {
	from string
	to string
}

// TODO
func getSwitchIds(topo *shimv1alpha1.Topology) map[string]int {
	res := map[string]int{}
	counter := 0
	for _, dev := range topo.Spec.Graph {
		if dev.DeviceType == shimv1alpha1.INC_SWITCH {
			res[dev.Name] = counter
			counter++
		}
	}
	return res
}

func dfs(G map[string]shimv1alpha1.NetworkDevice, target string, cur string, reversedPath *[]string, visited map[string]bool) {
	visited[cur] = true
	if cur == target {
		*reversedPath = append(*reversedPath, cur)
		return
	}
	dev := G[cur]
	for _, neigh := range dev.Links {
		if !visited[neigh.PeerName] {
			dfs(G, target, neigh.PeerName, reversedPath, visited)
			if len(*reversedPath) > 0 {
				*reversedPath = append(*reversedPath, cur)
				return
			}
		}
	}
}

type DependingEntries [T comparable] struct {
	Entries map[T][]incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry
} 

func newDependingEntries[T comparable]() *DependingEntries[T] {
	return &DependingEntries[T]{
		Entries: make(map[T][]incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry),
	}
}

func (d *DependingEntries[T]) addDependency(key T, entry incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry) {
	if e, ok := d.Entries[key]; ok {
		d.Entries[key] = append(e, entry)
	} else {
		d.Entries[key] = []incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry{entry}
	}
}

func (r *ExternalInNetworkTelemetryEndpointsReconciler) establishStuff(
		topo *shimv1alpha1.Topology,
		entries []incv1alpha1.ExternalInNetworkTelemetryEndpointsEntry,
		programName string) {
	// assumption: both src and target are either node/external devices, 
	// other devices are either INC or NET

	sourceName := "test-cluster-worker2" // w1 -> w3, w1 initiated, for now
	sinks := newDependingEntries[Edge]()
	sources := newDependingEntries[Edge]()
	transits := newDependingEntries[string]()
	switchIds := getSwitchIds(topo)
	G := shimutils.TopologyToGraph(topo)

	for _, entry := range entries {
		if entry.EntryStatus == incv1alpha1.EP_PENDING {
			if sourceName == entry.NodeName {
				// nothing to do
			} else {
				visited := map[string]bool{}
				reversedPath := []string{}
				dfs(G, sourceName, entry.NodeName, &reversedPath, visited)
				if len(reversedPath) == 0 {
					panic("no path")
				}
				i := 1
				for ; i < len(reversedPath); i++ {
					dev := G[reversedPath[i]]
					if dev.DeviceType == shimv1alpha1.INC_SWITCH /* && program == telemetry */ {
						break
					}
				}
				if i >= len(reversedPath) {
					// set failed as most likely topo changes or scheduler fup'ed
					continue
				}
				sourceDev := reversedPath[i]
				j := len(reversedPath) - 2
				for ; j >= i; j-- {
					dev := G[reversedPath[j]]
					if dev.DeviceType == shimv1alpha1.INC_SWITCH /* && program == telemetry */ {
						break
					}
				}
				sinkDev := reversedPath[j]
				for k := i + 1; k < j; k++ {
					dev := G[reversedPath[k]]
					if dev.DeviceType == shimv1alpha1.INC_SWITCH /* && program == telemetry */ {
						transits.addDependency(dev.Name, entry)
					}
				}
				sinks.addDependency(Edge{from: sinkDev, to: reversedPath[j+1]}, entry)
				sources.addDependency(Edge{from: reversedPath[i - 1], to: sourceDev}, entry)
			}
		}
	}



	_ = switchIds
}

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

func (r *ExternalInNetworkTelemetryEndpointsReconciler) buildEnableTelemetryRequestIfChangeDetected(
	ctx context.Context,
	endpoints *incv1alpha1.ExternalInNetworkTelemetryEndpoints,
) (*pbt.EnableTelemetryRequest, bool) {
	targetDevices := map[string]struct{}{}
	changed := false

	for _, entry := range endpoints.Spec.Entries {
		if entry.EntryStatus == incv1alpha1.EP_READY {
			targetDevices[entry.NodeName] = struct{}{}
		} else if entry.EntryStatus == incv1alpha1.EP_TERMINATING {
			changed = true
		} else if entry.EntryStatus == incv1alpha1.EP_PENDING {
			changed = true
			targetDevices[entry.NodeName] = struct{}{}
		}
	}
	if !changed {
		return nil, false
	}

	sourceNodeName, err := shimutils.GetNameOfNodeWithSname(ctx, r.Client, "w1")
	if err != nil {
		panic(err)
	}

	targetDeviceNames := make([]string, 0, len(targetDevices))
	for devName := range targetDevices {
		targetDeviceNames = append(targetDeviceNames, devName)
	}
	return &pbt.EnableTelemetryRequest{
		ProgramName: endpoints.Spec.ProgramName,
		CollectionId: endpoints.Name,
		CollectorNodeName: endpoints.Spec.CollectorNodeName,
		CollectorPort: 6000,
		Sources: &pbt.EnableTelemetryRequest_RawSources{
			RawSources: &pbt.RawTelemetryEntities{
				DeviceNames: []string{sourceNodeName},
			},
		},
		Targets: &pbt.EnableTelemetryRequest_RawTargets{
			RawTargets: &pbt.RawTelemetryEntities{
				DeviceNames: targetDeviceNames,
			},
		},
	}, true
}

func (r *ExternalInNetworkTelemetryEndpointsReconciler) withTelemetryClient(grpcAddr string, consumer func(pbt.TelemetryServiceClient) error) error {
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	return consumer(pbt.NewTelemetryServiceClient(conn))
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

	log.Info("Handling endpoints ---")

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
	_ = topo
	_ = shim

	if enableTelemetryRequest, stateChanged := r.buildEnableTelemetryRequestIfChangeDetected(ctx, endpoints); stateChanged {
		var resp *pbt.EnableTelemetryResponse
		err = r.withTelemetryClient(shim.Spec.SdnConfig.TelemetryServiceGrpcAddr, func(tsc pbt.TelemetryServiceClient) error {
			enableResp, err := tsc.EnableTelemetry(ctx, enableTelemetryRequest)
			if err != nil {
				return err
			}
			resp = enableResp
			return nil
		})	
		if err != nil {
			log.Error(err, "Failed to make enable telemetry request")
			return ctrl.Result{}, err
		}
		switch resp.TelemetryState {
		case pbt.TelemetryState_IN_PROGRESS:
			log.Info("Enabling telemetry is in progress")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		case pbt.TelemetryState_FAILED:
			log.Info("Failed to set telemetry")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		case pbt.TelemetryState_OK:
			for i := range endpoints.Spec.Entries {
				endpoints.Spec.Entries[i].EntryStatus = incv1alpha1.EP_READY
			}
			if err := r.Update(ctx, endpoints); err != nil {
				log.Error(err, "Failed to update endpoints even though request went through")	
				return ctrl.Result{}, err
			}
		default:
			panic("unexpected telemetry state")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExternalInNetworkTelemetryEndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.ExternalInNetworkTelemetryEndpoints{}).
		Complete(r)
}
