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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
)

const SDN_RETRY_PERIOD = 2 * time.Second

// InternalInNetworkTelemetryEndpointsReconciler reconciles a InternalInNetworkTelemetryEndpoints object
type InternalInNetworkTelemetryEndpointsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetryendpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetryendpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=internalinnetworktelemetryendpoints/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=collectors,verbs=get;list;watch

func (r *InternalInNetworkTelemetryEndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling internal endpoints")


	endpoints := &incv1alpha1.InternalInNetworkTelemetryEndpoints{}
	if err := r.Get(ctx, req.NamespacedName, endpoints); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	shim, err := shimutils.LoadSDNShim(ctx, r.Client)
	if err != nil {
		log.Error(err, "failed to load SDNShim config")
		return ctrl.Result{}, err
	}

	isMarkedForDeletion := endpoints.GetDeletionTimestamp() != nil
	if isMarkedForDeletion {
		if controllerutil.ContainsFinalizer(endpoints, INTERNAL_TELEMETRY_ENDPOINTS_FINALIZER) {
			shouldRetry := false
			err := r.withTelemetryClient(shim.Spec.SdnConfig.SdnGrpcAddr, func(tsc pbt.TelemetryServiceClient) error {
				sr, err := r.disableTelemetry(ctx, tsc, endpoints)
				shouldRetry = sr
				return err
			})
			if err != nil {
				log.Error(err, "failed to disable telemetry")
				return ctrl.Result{}, err
			}
			if shouldRetry {
				return ctrl.Result{RequeueAfter: SDN_RETRY_PERIOD}, nil
			} else {
				controllerutil.RemoveFinalizer(endpoints, INTERNAL_TELEMETRY_ENDPOINTS_FINALIZER)
				if err := r.Update(ctx, endpoints); err != nil {
					log.Error(err, "failed to remove finalizer for internal endpoints")
					return ctrl.Result{}, err
				}
			}
		}
		return ctrl.Result{}, nil
	}

	collector := &incv1alpha1.Collector{}
	collectorKey := types.NamespacedName{Name: endpoints.Spec.CollectorRef.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, collectorKey, collector); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "no collector")
			return ctrl.Result{}, err // TODO: disable telemetry if it was enabled
		}
		log.Error(err, "failed to load collector")
		return ctrl.Result{}, err
	}

	if !meta.IsStatusConditionTrue(collector.Status.Conditions, incv1alpha1.TypeAvailableCollector) {
		log.Info("collector not ready")
		return ctrl.Result{}, nil
	}

	perDeplEntities := map[string]map[string]*pbt.TunneledEntities{}
	changed := false
	for _, deplEndpoints := range endpoints.Spec.DeploymentEndpoints {
		entities, chngd := r.getEntitiesForEndpoints(deplEndpoints)
		changed = changed || chngd
		perDeplEntities[deplEndpoints.DeploymentName] = entities
	}
	if !changed {
		return ctrl.Result{}, nil
	}

	success, shouldRequeue := true, false
	err = r.withTelemetryClient(shim.Spec.SdnConfig.SdnGrpcAddr, func(tsc pbt.TelemetryServiceClient) error {
		for _, p := range utils.PairedDeployments(endpoints) {
			resp, err := r.sendConfigureTelemetryRequest(ctx, tsc, endpoints, collector, p.First, p.Second, perDeplEntities)
			if err != nil {
				return err
			}
			if resp.TelemetryState != pbt.TelemetryState_OK {
				success = false
				if resp.Description != nil {
					log.Info(*resp.Description)
				}
				if resp.TelemetryState == pbt.TelemetryState_IN_PROGRESS {
					shouldRequeue = true
				} else {
					break // probably no reason to configure the next one
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Error(err, "Failed to configure telemetry")
		return ctrl.Result{}, err
	}
	if shouldRequeue {
		return ctrl.Result{RequeueAfter: time.Second*5}, nil
	}
	if !success {
		// TODO: display some status that it couldn't have been set
		log.Info("SDN refused to configure telemetry")
		return ctrl.Result{RequeueAfter: time.Second*5}, nil
	}
	for i, eps := range endpoints.Spec.DeploymentEndpoints {
		newEntries := []incv1alpha1.InternalInNetworkTelemetryEndpointsEntry{}
		for _, entry := range eps.Entries {
			if entry.EntryStatus != incv1alpha1.EP_TERMINATING {
				entry.EntryStatus = incv1alpha1.EP_READY
				newEntries = append(newEntries, entry)
			}
		}
		endpoints.Spec.DeploymentEndpoints[i].Entries = newEntries
	}
	if err = r.Update(ctx, endpoints); err != nil {
		log.Error(err, "Failed to update endpoints even though requests went through")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InternalInNetworkTelemetryEndpointsReconciler) getEntitiesForEndpoints(
	deplEndpoints incv1alpha1.DeploymentEndpoints,
) (res map[string]*pbt.TunneledEntities, changed bool) {
	res = make(map[string]*pbt.TunneledEntities)
	changed = false
	for _, entry := range deplEndpoints.Entries {
		valid := false
		switch entry.EntryStatus {
		case incv1alpha1.EP_READY:
			valid = true
		case incv1alpha1.EP_TERMINATING:
			changed = true
		case incv1alpha1.EP_PENDING:
			valid = true
			changed = true
		default:
			panic("unexpected entry status")
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

func (r *InternalInNetworkTelemetryEndpointsReconciler) sendConfigureTelemetryRequest(
	ctx context.Context,
	client pbt.TelemetryServiceClient,
	endpoints *incv1alpha1.InternalInNetworkTelemetryEndpoints,
	collector *incv1alpha1.Collector,
	sourceDepl string,
	targetDepl string,
	perDeplEntities map[string]map[string]*pbt.TunneledEntities,
) (*pbt.ConfigureTelemetryResponse, error) {
	req := &pbt.ConfigureTelemetryRequest{
		IntentId: utils.BuildInternalIntentId(endpoints.Name, sourceDepl, targetDepl),
		CollectionId: endpoints.Spec.CollectionId,
		CollectorNodeName: collector.Status.NodeRef.Name,
		CollectorPort: *collector.Status.ReportingPort,
		Sources: &pbt.ConfigureTelemetryRequest_TunneledSources{
			TunneledSources: &pbt.TunneledTelemetryEntities{
				DeviceNamesWithEntities: perDeplEntities[sourceDepl],	
			},
		},
		Targets: &pbt.ConfigureTelemetryRequest_TunneledTargets{
			TunneledTargets: &pbt.TunneledTelemetryEntities{
				DeviceNamesWithEntities: perDeplEntities[targetDepl],	
			},
		},
	}
	return client.ConfigureTelemetry(ctx, req)
}

func (r *InternalInNetworkTelemetryEndpointsReconciler) disableTelemetry(
	ctx context.Context,
	client pbt.TelemetryServiceClient,
	endpoints *incv1alpha1.InternalInNetworkTelemetryEndpoints,
) (shouldRetry bool, err error) {
	shouldRetry = false
	// TODO: concurrently
	for _, p := range utils.PairedDeployments(endpoints) {
		resp, err := r.sendDisableTelemetryRequest(ctx, client, endpoints, p.First, p.Second)
		if err != nil {
			return true, err		
		}
		shouldRetry = shouldRetry || resp.ShouldRetryLater 
	}
	return shouldRetry, nil
}

func (r *InternalInNetworkTelemetryEndpointsReconciler) sendDisableTelemetryRequest(
	ctx context.Context,
	client pbt.TelemetryServiceClient,
	endpoints *incv1alpha1.InternalInNetworkTelemetryEndpoints,
	sourceDepl string,
	targetDepl string,
) (*pbt.DisableTelemetryResponse, error) {
	req := &pbt.DisableTelemetryRequest{
		IntentId: utils.BuildInternalIntentId(endpoints.Name, sourceDepl, targetDepl),
	}
	return client.DisableTelemetry(ctx, req)
}

func (r *InternalInNetworkTelemetryEndpointsReconciler) findEndpointsThatUseCollector(collectorRawObj client.Object) []reconcile.Request {
	collector := collectorRawObj.(*incv1alpha1.Collector)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
	defer cancel()
	endpointsList := &incv1alpha1.InternalInNetworkTelemetryEndpointsList{}
	// TODO: use field selector and add index on collectorRef
	listOpts := client.ListOptions{
		Namespace: collector.Namespace,
	}
	if err := r.List(ctx, endpointsList, &listOpts); err != nil {
		// we can't do much else, this can happen due to a connection issue
		fmt.Printf("Failed to list endpoints for collector %e", err)
		return []reconcile.Request{}
	}
	res := []reconcile.Request{}
	for _, ep := range endpointsList.Items {
		if ep.Spec.CollectorRef.Name == collector.Name {
			res = append(res, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&ep)})
		}
	}
	return res
}

func (r *InternalInNetworkTelemetryEndpointsReconciler) withTelemetryClient(grpcAddr string, consumer func(pbt.TelemetryServiceClient) error) error {
	// TODO: reuse short-lived connections (don't hang on SDN all the time)
	// use different goroutine to manage the client connection
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	return consumer(pbt.NewTelemetryServiceClient(conn))
}

// SetupWithManager sets up the controller with the Manager.
func (r *InternalInNetworkTelemetryEndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.InternalInNetworkTelemetryEndpoints{}).
		Watches(
			&source.Kind{Type: &incv1alpha1.Collector{}},
			handler.EnqueueRequestsFromMapFunc(r.findEndpointsThatUseCollector),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}
