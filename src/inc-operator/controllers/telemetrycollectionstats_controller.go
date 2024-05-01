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
	"fmt"
	"io"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	"github.com/Fl0k3n/k8s-inc/inc-operator/shimutils"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
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

	stats := &incv1alpha1.TelemetryCollectionStats{}
	if err := r.Get(ctx, req.NamespacedName, stats); err != nil {
		return ctrl.Result{}, nil
	}
	collector := &incv1alpha1.Collector{}
	collectorKey := types.NamespacedName{Name: stats.Spec.CollectorRef.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, collectorKey, collector); err != nil {
		// TODO do this properly, watch etc...
		log.Error(err, "collector not found")
		return ctrl.Result{RequeueAfter: 2*time.Second}, nil
	}
	if !meta.IsStatusConditionTrue(collector.Status.Conditions, incv1alpha1.TypeAvailableCollector) {
		log.Info("collector not ready")
		return ctrl.Result{RequeueAfter: 2*time.Second}, nil
	}
	collectorService := &v1.Service{}
	serviceKey := types.NamespacedName{Name: collector.Status.ApiServiceRef.Name, Namespace: req.Namespace}
	if err := r.Get(ctx, serviceKey, collectorService); err != nil {
		log.Error(err, "collector service not found")
		return ctrl.Result{RequeueAfter: 2*time.Second}, nil
	}
	if len(collectorService.Spec.Ports) == 0 {
		log.Info("collector service not ready")
		return ctrl.Result{RequeueAfter: 2*time.Second}, nil

	}
	apiAddress := fmt.Sprintf("http://%s:%d", collectorService.Spec.ClusterIP, int(collectorService.Spec.Ports[0].Port))
	handlersKey := types.NamespacedName{Name: stats.Spec.CollectorRef.Name, Namespace: req.Namespace}
	handlers, ok := r.collectorHandlers[handlersKey]
	if !ok {
		handlers = newCollectorStatHandlers(r.Client, r.Connector, apiAddress)
		r.collectorHandlers[handlersKey] = handlers
	}
	handlers.UpdateCollectorAddress(apiAddress)
	handlers.CreateOrUpdateStatHandler(req.NamespacedName, stats.Spec.CollectionId, stats.Spec.RefreshPeriodMillis)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
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
		Complete(r)
}

type collectionId = string

type CollectorStatHandlers struct {
	client client.Client
	connector *shimutils.TelemetryPluginConnector
	collectorApiAddress string
	handlers map[collectionId]*StatHandler
}

func newCollectorStatHandlers(
	client client.Client,
	connector *shimutils.TelemetryPluginConnector,
	collectorApiAddress string,
) *CollectorStatHandlers {
	return &CollectorStatHandlers{
		client: client,
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
	handler := newStatHandler(c.client, c.connector, resourceKey, 
	refreshPeriodMillis, c.collectorApiAddress, collectionId)
	c.handlers[collectionId] = handler
	handler.Run()
}

type StatHandler struct {
	client client.Client
	connector *shimutils.TelemetryPluginConnector
	resourceKey types.NamespacedName
	refreshPeriodMillis int
	collectorAddress string
	collectionId collectionId
	stopChan chan struct{}
}

func newStatHandler(
	client client.Client,
	connector *shimutils.TelemetryPluginConnector,
	resourceKey types.NamespacedName,
	refreshPeriodMillis int,
	collectorAddress string,
	collectionId collectionId,
) *StatHandler {
	return &StatHandler{
		client: client,
		connector: connector,
		resourceKey: resourceKey,
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
					if err != nil {
						collectionId = int(collectionIdResp.CollectionId)
					}
					return err
				})
				if err != nil {
					continue
				}
				rawStats, err := s.sendGetMetricsRequest(collectionId)
				if err != nil {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				stats := &incv1alpha1.TelemetryCollectionStats{}
				if err := s.client.Get(ctx, s.resourceKey, stats); err != nil {
					cancel()
					continue
				}
				stats.Status.RawStats = &rawStats
				s.client.Status().Update(ctx, stats); 
				cancel()
			}
		}
	}()
}

func (s *StatHandler) sendGetMetricsRequest(collectionId int) (string, error) {
	body := []byte(fmt.Sprintf(`{"collection_id": %d}`, collectionId))
	// curl --header "Content-Type: application/json" --request POST --data '{"collection_id": 0}' 10.244.1.2:8000
	r, err := http.Post(s.collectorAddress, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	metrics, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	return string(metrics), nil
}

func (s *StatHandler) Stop() {
	s.stopChan <- struct{}{}
}
