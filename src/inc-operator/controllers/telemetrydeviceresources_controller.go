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
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/inc-operator/api/v1alpha1"
	"github.com/Fl0k3n/k8s-inc/inc-operator/shimutils"
	pbt "github.com/Fl0k3n/k8s-inc/proto/sdn/telemetry"
	shimv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const CLUSTER_TELEMETRY_DEVICE_RESOURCES_NAME = "telemetry-device-resources"
const CAN_BE_SOURCE_THRESHOLD = 10
const UPDATE_STATUS_ATTEMPTS = 3

// TelemetryDeviceResourcesReconciler reconciles a TelemetryDeviceResources object
type TelemetryDeviceResourcesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=telemetrydeviceresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=telemetrydeviceresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=telemetrydeviceresources/finalizers,verbs=update

// returns error if creation (and fetch) failed
func (r *TelemetryDeviceResourcesReconciler) getOrCreateResources(
	ctx context.Context,
	shim *shimv1alpha1.SDNShim,
) (*incv1alpha1.TelemetryDeviceResources, error) {
	resourceKey := types.NamespacedName{Name: CLUSTER_TELEMETRY_DEVICE_RESOURCES_NAME, Namespace: shim.Namespace}
	res := &incv1alpha1.TelemetryDeviceResources{}
	err := r.Get(ctx, resourceKey, res)
	if err == nil {
		return res, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	res = &incv1alpha1.TelemetryDeviceResources{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceKey.Name,
			Namespace: resourceKey.Namespace,
		},
		Spec: incv1alpha1.TelemetryDeviceResourcesSpec{},
	}
	if err := r.Create(ctx, res); err != nil {
		return nil, err
	}
	// refetch to get correct version etc
	res = &incv1alpha1.TelemetryDeviceResources{}
	err = r.Get(ctx, resourceKey, res)
	return res, err
}

func (r *TelemetryDeviceResourcesReconciler) updateStatus(
	ctx context.Context,
	shim *shimv1alpha1.SDNShim,
	update *pbt.SourceCapabilityUpdate,
) error {
	res := []incv1alpha1.TelemetryDeviceResource{}
	for deviceName, remainingEntries := range update.RemainingSourceEndpoints {
		res = append(res, incv1alpha1.TelemetryDeviceResource{
			DeviceName: deviceName,
			CanBeSource: remainingEntries > CAN_BE_SOURCE_THRESHOLD,
		})
	}
	var err error = nil
	for i := 0; i < UPDATE_STATUS_ATTEMPTS; i++ {
		resources, er := r.getOrCreateResources(ctx, shim)
		if er != nil {
			return er
		}
		resources.Status.DeviceResources = res
		if err = r.Status().Update(ctx, resources); err == nil {
			return nil
		}
		if apierrors.IsNotFound(err) {
			time.Sleep(2 * time.Second)
			continue
		}
		if !apierrors.IsConflict(err) {
			return err
		}
	}
	return err
}

func (r *TelemetryDeviceResourcesReconciler) watchTelemetryUpdates(log logr.Logger) {
	for {
		var conn *grpc.ClientConn = nil
		var err error
		var client pbt.TelemetryServiceClient
		var stream pbt.TelemetryService_SubscribeSourceCapabilitiesClient = nil
		var grpcAddr string

		ctx := context.Background()
		shim, err := shimutils.LoadSDNShim(ctx, r.Client)
		if err != nil {
			log.Info("SDN shim unavailable")
			goto retry
		}
		grpcAddr = shim.Spec.SdnConfig.TelemetryServiceGrpcAddr
		conn, err = grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Info(fmt.Sprintf("Failed to connect to SDN telemetry plugin %e", err))
			goto retry
		}
		client = pbt.NewTelemetryServiceClient(conn)
		stream, err = client.SubscribeSourceCapabilities(ctx, &emptypb.Empty{})
		if err != nil {
			log.Info(fmt.Sprintf("Failed to watch resource updates %e", err))
			goto retry
		}
		log.Info("Watching Device Telemetry Resource updates")
		for {
			update, err := stream.Recv()
			if err == io.EOF {
				log.Info("Server closed stream")	
				goto retry
			}
			if err != nil {
				log.Error(err, "Recv failed")
				goto retry
			}
			if err := r.updateStatus(ctx, shim, update); err != nil {
				log.Error(err, "Failed to update resource status")
				goto retry
			}
		}
	retry:
		if conn != nil {
			conn.Close()
		}
		time.Sleep(5 * time.Second)
	}
}

func (r *TelemetryDeviceResourcesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	go r.watchTelemetryUpdates(mgr.GetLogger())
	return nil
}
