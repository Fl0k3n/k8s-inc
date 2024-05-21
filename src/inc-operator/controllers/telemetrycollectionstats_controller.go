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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	"github.com/Fl0k3n/k8s-inc/inc-operator/controllers/utils"
	"github.com/Fl0k3n/k8s-inc/inc-operator/shimutils"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
)

const (
	UNKNOWN_SWITCH_ID_DESCRIPTOR = "unknown"
	RECONCILIATION_FAILURE_REQUEUE_AFTER = 2*time.Second
	TELEMETRY_COLLECTION_STATS_FINALIZER = "inc.kntp.com/telemetry-stats-finalizer"
)

// TelemetryCollectionStatsReconciler reconciles a TelemetryCollectionStats object
type TelemetryCollectionStatsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Connector *shimutils.TelemetryPluginConnector
	collectorHandlers map[types.NamespacedName]*CollectorStatHandlers
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=telemetrycollectionstats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=telemetrycollectionstats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=telemetrycollectionstats/finalizers,verbs=update

func (r *TelemetryCollectionStatsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	retryLater := ctrl.Result{RequeueAfter: RECONCILIATION_FAILURE_REQUEUE_AFTER}

	// TODO: we also should keep info about collectionId and collectorRef for a given resource
	// so that when they change we can do a proper cleanup
	stats := &incv1alpha1.TelemetryCollectionStats{}
	if err := r.Get(ctx, req.NamespacedName, stats); err != nil {
		return ctrl.Result{}, nil
	}
	handlersKey := types.NamespacedName{Name: stats.Spec.CollectorRef.Name, Namespace: req.Namespace}
	handlers, hasCollectorHandlers := r.collectorHandlers[handlersKey]

	isMarkedForDeletion := stats.GetDeletionTimestamp() != nil
	hasFinalizer := controllerutil.ContainsFinalizer(stats, TELEMETRY_COLLECTION_STATS_FINALIZER)
	if isMarkedForDeletion {
		if hasFinalizer {
			if hasCollectorHandlers {
				if hasMoreHandlers := handlers.RemoveStatHandler(stats.Spec.CollectionId); !hasMoreHandlers {
					delete(r.collectorHandlers, handlersKey)
				}
			}
			controllerutil.RemoveFinalizer(stats, TELEMETRY_COLLECTION_STATS_FINALIZER)
			if err := r.Update(ctx, stats); err != nil {
				log.Error(err, "failed to remove finalizer for telemetry stats")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	} else if !hasFinalizer {
		controllerutil.AddFinalizer(stats, TELEMETRY_COLLECTION_STATS_FINALIZER)
		if err := r.Update(ctx, stats); err != nil {
			log.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	topo, err := shimutils.LoadTopology(ctx, r.Client)
	if err != nil {
		log.Error(err, "topology unavailable")
		return retryLater, nil
	}
	collector := &incv1alpha1.Collector{}
	collectorKey := types.NamespacedName{Name: stats.Spec.CollectorRef.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, collectorKey, collector); err != nil {
		if apierrors.IsNotFound(err) {
			if hasCollectorHandlers {
				log.Info("collector removed")
				handlers.StopAll()
				delete(r.collectorHandlers, handlersKey)
			}
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to load collector")
		return ctrl.Result{}, err
	}
	if !meta.IsStatusConditionTrue(collector.Status.Conditions, incv1alpha1.TypeAvailableCollector) {
		if hasCollectorHandlers {
			handlers.StopAll()
			delete(r.collectorHandlers, handlersKey)
		}
		return ctrl.Result{}, nil
	}
	collectorService := &v1.Service{}
	serviceKey := types.NamespacedName{Name: collector.Status.ApiServiceRef.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, serviceKey, collectorService); err != nil {
		log.Error(err, "collector service not found")
		return retryLater, nil
	}
	if len(collectorService.Spec.Ports) == 0 {
		log.Info("collector service not ready")
		return retryLater, nil
	}
	apiAddress := fmt.Sprintf("http://%s:%d", collectorService.Spec.ClusterIP, int(collectorService.Spec.Ports[0].Port))
	if !hasCollectorHandlers {
		// TODO handle changing topology
		handlers = newCollectorStatHandlers(r.Client, r.Connector, topo, apiAddress)
		r.collectorHandlers[handlersKey] = handlers
	}
	handlers.UpdateCollectorAddress(apiAddress)
	handlers.CreateOrUpdateStatHandler(req.NamespacedName, stats.Spec.CollectionId, stats.Spec.RefreshPeriodMillis)
	return ctrl.Result{}, nil
}

func (r *TelemetryCollectionStatsReconciler) findStatsThatUseCollector(collectorRawObj client.Object) []reconcile.Request {
	collector := collectorRawObj.(*incv1alpha1.Collector)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
	defer cancel()
	statsList := &incv1alpha1.TelemetryCollectionStatsList{}
	// TODO: use field selector and add index on collectorRef
	listOpts := client.ListOptions{
		Namespace: collector.Namespace,
	}
	if err := r.List(ctx, statsList, &listOpts); err != nil {
		// we can't do much else, this can happen due to a connection issue
		fmt.Printf("Failed to list stats for collector %e", err)
		return []reconcile.Request{}
	}
	res := []reconcile.Request{}
	for _, stats := range statsList.Items {
		if stats.Spec.CollectorRef.Name == collector.Name {
			res = append(res, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&stats)})
		}
	}
	return res
}

func (r *TelemetryCollectionStatsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.collectorHandlers = make(map[types.NamespacedName]*CollectorStatHandlers)
	initChan := make(chan event.GenericEvent)
	log := mgr.GetLogger()
	go func() {
		for {
			statsList := &incv1alpha1.TelemetryCollectionStatsList{}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := r.Client.List(ctx, statsList); err != nil {
				cancel()
				log.Error(err, "failed to list existing telemetry collection stats")
				time.Sleep(2 * time.Second)
				continue
			}
			cancel()
			for i := range statsList.Items {
				initChan <- event.GenericEvent{Object: &statsList.Items[i]}
			}
			break
		}
	}()
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.TelemetryCollectionStats{}).
		Watches(&source.Channel{Source: initChan}, &handler.EnqueueRequestForObject{}).
		Watches(
			&source.Kind{Type: &incv1alpha1.Collector{}},
			handler.EnqueueRequestsFromMapFunc(r.findStatsThatUseCollector),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

type collectionId = string

// not thread-safe, don't use concurrent reconciling
type CollectorStatHandlers struct {
	client client.Client
	topo shimutils.TopologyGraph
	switchIdToName map[int]string
	connector *shimutils.TelemetryPluginConnector
	collectorApiAddress string
	handlers map[collectionId]*StatHandler
}

func newCollectorStatHandlers(
	client client.Client,
	connector *shimutils.TelemetryPluginConnector,
	topo *shimv1alpha1.Topology,
	collectorApiAddress string,
) *CollectorStatHandlers {
	return &CollectorStatHandlers{
		client: client,
		topo: shimutils.TopologyToGraph(topo),
		switchIdToName: shimutils.GetSwitchIdToNameMapping(topo),
		connector: connector,
		collectorApiAddress: collectorApiAddress,
		handlers: map[collectionId]*StatHandler{},
	}
}

func (c *CollectorStatHandlers) UpdateCollectorAddress(address string) {
	if c.collectorApiAddress == address {
		return
	}
	c.collectorApiAddress = address
	for cid, statHandler := range c.handlers {
		c.CreateOrUpdateStatHandler(statHandler.resourceKey, cid, statHandler.refreshPeriodMillis)
	}
}

func (c *CollectorStatHandlers) CreateOrUpdateStatHandler(
	resourceKey types.NamespacedName,
	collectionId collectionId,
	refreshPeriodMillis int,
) {
	if handler, ok := c.handlers[collectionId]; ok {
		if handler.refreshPeriodMillis == refreshPeriodMillis &&
		   handler.resourceKey == resourceKey && 
		   handler.collectorAddress == c.collectorApiAddress {
			return
		}
		handler.Stop()
	} 	
	handler := newStatHandler(c.client, c.connector, c.topo, resourceKey,
		c.switchIdToName, refreshPeriodMillis, c.collectorApiAddress, collectionId)
	c.handlers[collectionId] = handler
	handler.Run()
}

func (c *CollectorStatHandlers) RemoveStatHandler(collectionId collectionId) (hasMoreHandlers bool) {
	if handler, ok := c.handlers[collectionId]; ok {
		handler.Stop()
		delete(c.handlers, collectionId)
	}
	return len(c.handlers) > 0
}

func (c *CollectorStatHandlers) StopAll() {
	for _, handler := range c.handlers {
		handler.Stop()
	}
	c.handlers = map[string]*StatHandler{}
}

type StatHandler struct {
	client client.Client
	connector *shimutils.TelemetryPluginConnector
	topo shimutils.TopologyGraph
	resourceKey types.NamespacedName
	switchIdToName map[int]string
	refreshPeriodMillis int
	collectorAddress string
	collectionId collectionId
	stopChan chan struct{}
}

func newStatHandler(
	client client.Client,
	connector *shimutils.TelemetryPluginConnector,
	topo shimutils.TopologyGraph,
	resourceKey types.NamespacedName,
	switchIdToName map[int]string,
	refreshPeriodMillis int,
	collectorAddress string,
	collectionId collectionId,
) *StatHandler {
	return &StatHandler{
		client: client,
		connector: connector,
		topo: topo,
		resourceKey: resourceKey,
		switchIdToName: switchIdToName,
		refreshPeriodMillis: refreshPeriodMillis,
		collectorAddress: collectorAddress,
		collectionId: collectionId,
		stopChan: make(chan struct{}),
	}
}

// runs in separate goroutine
func (s *StatHandler) Run() {
	duration := time.Duration(s.refreshPeriodMillis) * time.Millisecond
	ticker := time.NewTicker(time.Duration(duration))
	logPath := "/home/flok3n/develop/k8s_inc_analysis/data/new_http_tcp3/logs%d.json"
	i := 0
	go func() {
		for {
			select {
			case <-s.stopChan:
				ticker.Stop()
				return
			case <-ticker.C:
				var collectionId int
				err := s.connector.WithTelemetryClient(func(client pbt.TelemetryServiceClient) error {
					ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
					defer cancel()
					collectionIdResp, err := client.GetCollectionId(ctx, &pbt.GetCollectionIdRequest{CollectionId: s.collectionId})
					if err == nil {
						collectionId = int(collectionIdResp.CollectionId)
					}
					return err
				})
				if err != nil {
					continue
				}
				metrics, err := s.sendGetMetricsRequest(collectionId)
				if err != nil {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				stats := &incv1alpha1.TelemetryCollectionStats{}
				if err := s.client.Get(ctx, s.resourceKey, stats); err != nil {
					cancel()
					continue
				}
				stats.Status.MetricsSummary = s.convertToK8sRepr(metrics)
				
				// TODO: remove this, it's for eval only
				data, _ := json.Marshal(stats.Status.MetricsSummary)
				os.WriteFile(fmt.Sprintf(logPath, i), data, 0644)
				i += 1

				s.client.Status().Update(ctx, stats)
				cancel()
			}
		}
	}()
}

func (s *StatHandler) convertToK8sRepr(metrics *utils.MetricsSummary) *incv1alpha1.MetricsSummary {
	switchMetrics := make([]incv1alpha1.SwitchMetrics, len(metrics.WindowMetrics.DeviceMetrics))
	for i, m := range metrics.WindowMetrics.DeviceMetrics {
		switchName, ok := s.switchIdToName[m.SwitchID]
		if !ok {
			switchName = UNKNOWN_SWITCH_ID_DESCRIPTOR
		}
		portMetrics := make([]incv1alpha1.PortMetrics, len(m.PortMetrics))
		links := []shimv1alpha1.Link{}
		if switchNode, ok := s.topo[switchName]; ok {
			links = switchNode.Links
		} 
		for j, pm := range m.PortMetrics {
			neighIdx := pm.EgressPortID - 1
			targetName := UNKNOWN_SWITCH_ID_DESCRIPTOR
			if neighIdx >= 0 && neighIdx < len(links) {
				targetName = links[neighIdx].PeerName
			}
			portMetrics[j] = incv1alpha1.PortMetrics{
				TargetDeviceName: targetName,
				NumberPackets: pm.NumPackets,
				AverageLatencyMicroS: pm.AverageLatencyMicroS,
				AverageQueueFillState: pm.AverageQueueFillState,
				OnePercentileSlowestLatencyMicroS: pm.OnePercentileSlowestLatency,
				OnePercentileLargestQueueFillState: pm.OnePercentileLargestQueueFill,
			}
		}
		switchMetrics[i] = incv1alpha1.SwitchMetrics{
			DeviceName: switchName,
			PortMetrics: portMetrics,
		}
	}
	return &incv1alpha1.MetricsSummary{
		TotalReports: metrics.TotalReports,
		TimeWindowsSeconds: metrics.TimeWindowsSeconds,
		CreatedAt: time.Now().Unix(),
		WindowMetrics: incv1alpha1.Metrics{
			NumberCollectedReports: metrics.WindowMetrics.CollectedReports,
			AveragePathLatencyMicroS: metrics.WindowMetrics.AveragePathLatencyMicroS,
			OnePercentileSlowestPathLatencyMicroS: metrics.WindowMetrics.OnePercentileSlowestPathLatencyMicroS,
			DeviceMetrics: switchMetrics,
		},
	}
}

func (s *StatHandler) sendGetMetricsRequest(collectionId int) (*utils.MetricsSummary, error) {
	body := []byte(fmt.Sprintf(`{"collection_id": %d}`, collectionId))
	// curl --header "Content-Type: application/json" --request POST --data '{"collection_id": 0}' 10.244.1.2:8000
	r, err := http.Post(s.collectorAddress, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	metrics, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	parsed := &utils.MetricsSummary{}
	err = json.Unmarshal(metrics, parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func (s *StatHandler) Stop() {
	s.stopChan <- struct{}{}
}
