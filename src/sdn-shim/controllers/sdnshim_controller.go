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
	"sync"
	"time"

	sets "github.com/hashicorp/go-set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"

	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
)

var TOPOLOGY_KEY = types.NamespacedName{Name: "cluster-topo", Namespace: "default"}
const SDN_CONN_RETRY_PERIOD = 1*time.Second
var SDN_KEEP_ALIVE_PARAMS = keepalive.ClientParameters{
	Time:                15 * time.Second,
	Timeout:             5 * time.Second,
	PermitWithoutStream: false,
}

type SDNShimReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	sdnConnectorLock sync.Mutex
	ShimForClient *incv1alpha1.SDNShim
	sdnClient pb.SdnFrontendClient
	sdnConn *grpc.ClientConn
	networkUpdatesChan chan event.GenericEvent
	watcherStopChan chan struct{}
}

//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=sdnshims/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=topologies/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=incswitches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=incswitches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=incswitches/finalizers,verbs=update
//+kubebuilder:rbac:groups=inc.kntp.com,resources=p4programs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=p4programs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=p4programs/finalizers,verbs=update

func (r *SDNShimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	shimConfig := &incv1alpha1.SDNShim{}
	err := r.Get(ctx, req.NamespacedName, shimConfig)
	if err != nil {
		log.Info("SdnShim not found")
		return ctrl.Result{}, nil
	}

	if shimConfig.Status.Conditions == nil || len(shimConfig.Status.Conditions) == 0 {
		meta.SetStatusCondition(&shimConfig.Status.Conditions, metav1.Condition{
			Type: incv1alpha1.ConditionTypeSdnConnected,
			Status: metav1.ConditionFalse,
			Reason: "Reconciling",
			Message: "Starting reconciliation",
		})
		meta.SetStatusCondition(&shimConfig.Status.Conditions, metav1.Condition{
			Type: incv1alpha1.ConditionTypeNetworkReconciled,
			Status: metav1.ConditionFalse,
			Reason: "Reconciling",
			Message: "Starting reconciliation",
		})
		if err := r.Status().Update(ctx, shimConfig); err != nil {
			log.Error(err, "Failed to update shim status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	r.sdnConnectorLock.Lock()
	sdnClient := r.sdnClient
	r.sdnConnectorLock.Unlock()

	shouldRunClient := false
	if sdnClient == nil {
		shouldRunClient = true
	} else {
		if r.ShimForClient.Spec.SdnConfig.SdnGrpcAddr != shimConfig.Spec.SdnConfig.SdnGrpcAddr {
			r.closeSdnClient()
			shouldRunClient = true
		}
	}
	if shouldRunClient {
		if err := r.runSdnClientForShim(ctx, shimConfig); err != nil {
			log.Error(err, "Failed to connect to SDN controller")
			meta.SetStatusCondition(&shimConfig.Status.Conditions, metav1.Condition{
				Type: incv1alpha1.ConditionTypeSdnConnected,
				Status: metav1.ConditionFalse,
				Reason: "Reconciling",
				Message: "Failed to establish connection",
			})
			if err := r.Status().Update(ctx, shimConfig); err != nil {
				log.Error(err, "Failed to update shim status")
			}
			return ctrl.Result{RequeueAfter: SDN_CONN_RETRY_PERIOD}, nil
		}
		meta.SetStatusCondition(&shimConfig.Status.Conditions, metav1.Condition{
			Type: incv1alpha1.ConditionTypeSdnConnected,
			Status: metav1.ConditionTrue,
			Reason: "Reconciling",
			Message: "SDN connected",
		})
		if err := r.Status().Update(ctx, shimConfig); err != nil {
			log.Error(err, "Failed to update shim status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	sdnTopo, err := sdnClient.GetTopology(ctx, &emptypb.Empty{})
	if err != nil {
		r.closeSdnClient()
		log.Error(err, "Failed to connect to fetch topology from SDN")
		return ctrl.Result{Requeue: true}, nil
	}
	topo := r.buildClusterTopoCR(sdnTopo)

	storedTopo := &incv1alpha1.Topology{}
	if err := r.Get(ctx, TOPOLOGY_KEY, storedTopo); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, topo); err != nil {
				log.Error(err, "Failed to create topology CR")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	} else if r.topologyChanged(storedTopo, topo) {
		log.Info("Topology changed, updating")
		storedTopo.Spec.Graph = topo.Spec.Graph
		if err := r.Client.Update(ctx, storedTopo); err != nil {
			return ctrl.Result{}, err
		}
	}

	switchList := &incv1alpha1.IncSwitchList{}
	if err := r.List(ctx, switchList); err != nil {
		log.Error(err, "Failed to list incswitches")
		return ctrl.Result{}, err
	}
	presentSwitchLut := map[string]*incv1alpha1.IncSwitch{}
	presentSwitchNames := sets.New[string](0)
	for i, item := range switchList.Items {
		presentSwitchNames.Insert(item.Name)
		presentSwitchLut[item.Name] = &switchList.Items[i]
	}

	desiredSwitchNames := sets.New[string](0)
	for _, dev := range storedTopo.Spec.Graph {
		if dev.DeviceType == incv1alpha1.INC_SWITCH {
			desiredSwitchNames.Insert(dev.Name)
		}
	}
	switchesToAdd := desiredSwitchNames.Difference(presentSwitchNames)
	switchesToCheck := desiredSwitchNames.Intersect(presentSwitchNames)
	switchesToRemove := presentSwitchNames.Difference(desiredSwitchNames)
	
	for _, swName := range switchesToRemove.Slice() {
		if err := r.Client.Delete(ctx, presentSwitchLut[swName]); err != nil {
			log.Error(err, "failed to delete incSwitch")
			return ctrl.Result{}, err
		}
	}
	switchDetailsResp, err := sdnClient.GetSwitchDetails(ctx, &pb.SwitchNames{
		Names: switchesToAdd.Union(switchesToCheck).Slice(),
	})
	if err != nil {
		log.Error(err, "Failed to fetch switch details from SDN")
		statusCode := status.Code(err)
		if statusCode == codes.NotFound || statusCode == codes.InvalidArgument {
			return ctrl.Result{Requeue: true}, nil
		}
		r.closeSdnClient()
		return ctrl.Result{Requeue: true}, nil
	}
	desiredProgramNames := sets.New[string](0)
	for _, swName := range switchesToCheck.Slice() {
		if details, ok := switchDetailsResp.Details[swName]; ok {
			desiredProgramNames.Insert(details.InstalledProgram)
			storedDetails := presentSwitchLut[swName]
			if details.Arch != storedDetails.Spec.Arch || details.InstalledProgram != storedDetails.Spec.ProgramName {
				storedDetails.Spec.Arch = details.Arch
				storedDetails.Spec.ProgramName = details.InstalledProgram
				if err := r.Client.Update(ctx, storedDetails); err != nil {
					log.Error(err, "Failed to update incswitch")
					return ctrl.Result{}, err
				}
			}
		} else {
			log.Info(fmt.Sprintf("Unexpected SDN response, switch %s was queried but not returned", swName))
		}
	}
	for _, swName := range switchesToAdd.Slice() {
		if details, ok := switchDetailsResp.Details[swName]; ok {
			desiredProgramNames.Insert(details.InstalledProgram)
			if err := r.Client.Create(ctx, r.buildIncSwitchCR(details)); err != nil {
				log.Error(err, "Failed to create INC switch CR")
				return ctrl.Result{}, err
			}
		} else {
			log.Info(fmt.Sprintf("Unexpected SDN response, switch %s was queried but not returned", swName))
		}
	}

	programList := &incv1alpha1.P4ProgramList{}
	if err := r.Client.List(ctx, programList); err != nil {
		log.Error(err, "Failed to list p4 programs")
		return ctrl.Result{}, err
	}
	storedProgramNames := sets.New[string](len(programList.Items))
	programLut := map[string]*incv1alpha1.P4Program{}
	for i, p := range programList.Items {
		programLut[p.Name] = &programList.Items[i]
		storedProgramNames.Insert(p.Name)
	}
	programsToAdd := desiredProgramNames.Difference(storedProgramNames)
	programsToCheck := desiredProgramNames.Intersect(storedProgramNames)
	programsToRemove := storedProgramNames.Difference(desiredProgramNames)
	for _, programName := range programsToRemove.Slice() {
		if err := r.Delete(ctx, programLut[programName]); err != nil {
			log.Error(err, "failed to delete p4 program")
			return ctrl.Result{}, err
		}
	}
	desiredPrograms := map[string]*incv1alpha1.P4Program{}
	for _, programName := range programsToAdd.Union(programsToCheck).Slice() { 
		programDetailsResp, err := sdnClient.GetProgramDetails(ctx, &pb.ProgramDetailsRequest{
			ProgramName: programName,
		})
		if err != nil {
			log.Error(err, "Failed to fetch p4 program details from SDN")
			statusCode := status.Code(err)
			if statusCode == codes.NotFound {
				return ctrl.Result{Requeue: true}, nil
			}
			r.closeSdnClient()
			return ctrl.Result{Requeue: true}, nil
		}
		desiredPrograms[programName] = r.buildP4ProgramCR(programName, programDetailsResp)
	}
	for _, programName := range programsToAdd.Slice() {
		if err := r.Create(ctx, desiredPrograms[programName]); err != nil {
			log.Error(err, "Failed to create p4program")					
			return ctrl.Result{}, err
		}
	}
	for _, programName := range programsToCheck.Slice() {
		current, desired := programLut[programName], desiredPrograms[programName]
		if !slices.Equal(current.Spec.ImplementedInterfaces, desired.Spec.ImplementedInterfaces) {
			if err := r.Update(ctx, desired); err != nil {
				log.Error(err, "Failed to update p4program")
				return ctrl.Result{}, err
			}
		}
	}
	meta.SetStatusCondition(&shimConfig.Status.Conditions, metav1.Condition{
		Type: incv1alpha1.ConditionTypeNetworkReconciled,
		Status: metav1.ConditionTrue,
		Reason: "Reconciling",
		Message: "Network state reconciled",
	})
	if err := r.Status().Update(ctx, shimConfig); err != nil {
		log.Error(err, "Failed to update shim status")
		return ctrl.Result{}, err
	}
	log.Info("Reconciled SDNShim")
	return ctrl.Result{}, nil
}

func (r *SDNShimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.sdnConnectorLock = sync.Mutex{}
	r.networkUpdatesChan = make(chan event.GenericEvent)
	log := mgr.GetLogger()
	triggerRestoringInternalStateIfShimsExisted := func () {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			shims := &incv1alpha1.SDNShimList{}
			if err := r.Client.List(ctx, shims); err != nil {
				cancel()
				log.Error(err, "failed to list existing shims during init")
				time.Sleep(2*time.Second)
				continue
			}
			cancel()
			for i := range shims.Items {
				r.networkUpdatesChan <- event.GenericEvent{Object: &shims.Items[i]}
			}
			return
		}
	}
	go triggerRestoringInternalStateIfShimsExisted()
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&source.Channel{Source: r.networkUpdatesChan},
				&handler.EnqueueRequestForObject{}).
		For(&incv1alpha1.SDNShim{}).
		Complete(r)
}

func deviceTypeFromProto(dt pb.DeviceType) incv1alpha1.DeviceType {
	switch dt {
	case pb.DeviceType_EXTERNAL:   return incv1alpha1.EXTERNAL
	case pb.DeviceType_HOST:       return incv1alpha1.NODE
	case pb.DeviceType_NET: 	   return incv1alpha1.NET
	case pb.DeviceType_INC_SWITCH: return incv1alpha1.INC_SWITCH
	default:
		panic("unsupported device type")
	}
}

func (r *SDNShimReconciler) buildClusterTopoCR(topo *pb.TopologyResponse) *incv1alpha1.Topology {
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
			DeviceType: deviceTypeFromProto(dev.DeviceType),
			Links: links,
		}
	}
	
	res := &incv1alpha1.Topology{
		ObjectMeta: metav1.ObjectMeta{
			Name: TOPOLOGY_KEY.Name,
			Namespace: TOPOLOGY_KEY.Namespace,
		},
		Spec: incv1alpha1.TopologySpec{
			Graph: graph,
		},
	}
	return res
}

func (r *SDNShimReconciler) topologyChanged(previous *incv1alpha1.Topology, current *incv1alpha1.Topology) bool {
	if len(previous.Spec.Graph) != len(current.Spec.Graph) {
		return true
	}
	// we assume that order shouldn't change
	for i := range previous.Spec.Graph {
		prev, cur := previous.Spec.Graph[i], current.Spec.Graph[i]
		if prev.Name != cur.Name || prev.DeviceType != cur.DeviceType || len(prev.Links) != len(cur.Links) {
			return true
		}
		for j := range prev.Links {
			l1, l2 := prev.Links[j], cur.Links[j]
			if l1.PeerName != l2.PeerName {
				return true
			}
		}
	}
	return false
}

func (r *SDNShimReconciler) buildIncSwitchCR(details *pb.SwitchDetails) *incv1alpha1.IncSwitch {
	res := &incv1alpha1.IncSwitch{
		ObjectMeta: metav1.ObjectMeta{
			Name: details.Name,
			Namespace: TOPOLOGY_KEY.Namespace,
		},
		Spec: incv1alpha1.IncSwitchSpec{
			Arch: details.Arch,
			ProgramName: details.InstalledProgram,
		},
	}
	return res
}

func (r *SDNShimReconciler) buildP4ProgramCR(programName string, details *pb.ProgramDetailsResponse) *incv1alpha1.P4Program {
	return &incv1alpha1.P4Program{
		ObjectMeta: ctrl.ObjectMeta{
			Name: programName,
			Namespace: TOPOLOGY_KEY.Namespace,
		},
		Spec: incv1alpha1.P4ProgramSpec{
			ImplementedInterfaces: details.ImplementedInterfaces,
			Artifacts: []incv1alpha1.ProgramArtifacts{},
		},
	}
}


func (r *SDNShimReconciler) runSdnClientForShim(ctx context.Context, shim *incv1alpha1.SDNShim) error {
	r.sdnConnectorLock.Lock()
	defer r.sdnConnectorLock.Unlock()
	conn, err := grpc.Dial(
		shim.Spec.SdnConfig.SdnGrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(SDN_KEEP_ALIVE_PARAMS),
	)
	if err != nil {
		return err
	}
	client := pb.NewSdnFrontendClient(conn)
	changesStream, err := client.SubscribeNetworkChanges(ctx, &emptypb.Empty{})
	if err != nil {
		conn.Close()
		return err
	}
	r.ShimForClient = shim
	r.sdnConn = conn
	r.sdnClient = client
	r.watcherStopChan = make(chan struct{})
	go r.watchSdnNetworkUpdates(shim, changesStream, r.watcherStopChan)
	return nil
}

func (r *SDNShimReconciler) closeSdnClient() {
	r.sdnConnectorLock.Lock()
	defer r.sdnConnectorLock.Unlock()
	close(r.watcherStopChan)
	r.sdnConn.Close()
	r.ShimForClient = nil
	r.sdnConn = nil
	r.sdnClient = nil
}

func (r *SDNShimReconciler) watchSdnNetworkUpdates(
	shim *incv1alpha1.SDNShim,
	changesStream pb.SdnFrontend_SubscribeNetworkChangesClient,
	stopChan chan struct{},
) {
	for {
		_, err := changesStream.Recv()
		select {
		case _, open := <- stopChan:
			if !open {
				return
			}
		default:
			// continue
		}
		if err != nil {
			r.sdnConnectorLock.Lock()
			// if these conditions do not hold then reconciliation loop already
			// called close on behalf of this goroutine and we shouldn't do it again
			// if they do hold, this goroutine is the first entity that encountered some
			// connection related error (e.g. ping timeout) and we should notify reconciliation
			// loop to deal with this issue somehow (e.g. by reconnecting, changing status, etc...)
			if r.ShimForClient != nil &&
					client.ObjectKeyFromObject(shim) == client.ObjectKeyFromObject(r.ShimForClient) &&
					shim.ResourceVersion == r.ShimForClient.ResourceVersion {
				r.sdnConn.Close()
				r.networkUpdatesChan <- event.GenericEvent{Object: shim}
			}
			r.sdnConnectorLock.Unlock()
			return
		}
		r.networkUpdatesChan <- event.GenericEvent{Object: shim}
	}
}
