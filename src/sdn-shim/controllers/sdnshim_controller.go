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

	sets "github.com/hashicorp/go-set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	incv1alpha1 "github.com/Fl0k3n/k8s-inc/sdn-shim/api/v1alpha1"

	pb "github.com/Fl0k3n/k8s-inc/proto/sdn"
)

// SDNShimReconciler reconciles a SDNShim object

const ManagedClusterTopologyName = "cluster-topo"

type SDNShimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	sdnClient pb.SdnFrontendClient
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


func (r *SDNShimReconciler) runSdnClient(grpcAddr string) error {
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	r.sdnClient = pb.NewSdnFrontendClient(conn)
	return nil
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
			DeviceType: deviceTypeFromProto(dev.DeviceType),
			Links: links,
		}
	}
	
	res := &incv1alpha1.Topology{
		ObjectMeta: metav1.ObjectMeta{
			Name: ManagedClusterTopologyName,
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

func (r *SDNShimReconciler) buildIncSwitchCR(details *pb.SwitchDetails, req ctrl.Request) *incv1alpha1.IncSwitch {
	res := &incv1alpha1.IncSwitch{
		ObjectMeta: metav1.ObjectMeta{
			Name: details.Name,
			Namespace: req.Namespace,
		},
		Spec: incv1alpha1.IncSwitchSpec{
			Arch: details.Arch,
			ProgramName: details.InstalledProgram,
		},
		Status: incv1alpha1.IncSwitchStatus{
			InstalledProgram: details.InstalledProgram,
		},
	}
	return res
}

func (r *SDNShimReconciler) reconcileTopology(ctx context.Context, req ctrl.Request) (topo *incv1alpha1.Topology ,changed bool, err error) {
	topos := &incv1alpha1.TopologyList{}
	if err := r.List(ctx, topos); err != nil {
		return nil, false, err
	}
	if topos.Items == nil || len(topos.Items) == 0 {
		topo, err := r.sdnClient.GetTopology(ctx, &emptypb.Empty{})
		if err != nil {
			return nil, false, err
		}
		topoCr := r.buildClusterTopoCR(topo, req)
		if err := r.Client.Create(ctx, topoCr); err != nil {
			log.FromContext(ctx).Error(err, "Failed to create CR")
			return nil, false, err
		}
		return topoCr, true, nil
	} 
	log.FromContext(ctx).Info("Topo exists, TODO reconcile it if it may change") // TODO
	return &topos.Items[0], false, nil
}

func (r *SDNShimReconciler) reconcileP4Programs(ctx context.Context, req ctrl.Request, programNames *sets.Set[string]) error {
	// TODO maybe move it to p4program controller and watch for incswitch changes there
	log := log.FromContext(ctx)
	for _, programName := range programNames.Slice() {
		resourceKey := types.NamespacedName{Name: programName, Namespace: req.Namespace}
		p4program := &incv1alpha1.P4Program{}
		if err := r.Get(ctx, resourceKey, p4program); err != nil {
			if apierrors.IsNotFound(err) {
				programDetailsResp, err := r.sdnClient.GetProgramDetails(ctx, &pb.ProgramDetailsRequest{
					ProgramName: programName,
				})
				if err != nil {
					return err
				}
				p4program = &incv1alpha1.P4Program{
					ObjectMeta: ctrl.ObjectMeta{
						Name: programName,
						Namespace: req.Namespace,
					},
					Spec: incv1alpha1.P4ProgramSpec{
						ImplementedInterfaces: programDetailsResp.ImplementedInterfaces,
						Artifacts: []incv1alpha1.ProgramArtifacts{},
					},
				}
				if err := r.Create(ctx, p4program); err != nil {
					log.Error(err, "Failed to create p4program")					
					return err
				}
			} else {
				log.Error(err, "Failed to get p4program")
				return err
			}
		} else {
			// TODO check if program state matches
			_ = p4program
		}
	}
	return nil
}

func (r *SDNShimReconciler) reconcileIncSwitches(ctx context.Context, req ctrl.Request, topo *incv1alpha1.Topology) error {
	log := log.FromContext(ctx)
	switchList := &incv1alpha1.IncSwitchList{}
	if err := r.List(ctx, switchList); err != nil {
		return err
	}
	desiredSwitchNames := make([]string, 0)
	for _, dev := range topo.Spec.Graph {
		if dev.DeviceType == incv1alpha1.INC_SWITCH {
			desiredSwitchNames = append(desiredSwitchNames, dev.Name)
		}
	}
	switchesToRemove := make([]string, 0)
	switchesToAdd := make([]string, 0)
	switchesToCheckState := make([]string, 0)

	if switchList.Items == nil || len(switchList.Items) == 0 {
		if len(desiredSwitchNames) == 0 {
			return nil
		}
		switchesToAdd = desiredSwitchNames
	} else {
		// TODO cache in a faster struture if there can be many
		for _, presentItem := range switchList.Items {
			if slices.Contains(desiredSwitchNames, presentItem.Name) {
				switchesToCheckState = append(switchesToCheckState, presentItem.Name)
			} else {
				switchesToRemove = append(switchesToRemove, presentItem.Name)
			}
		}
		for _, desiredItem := range desiredSwitchNames {
			found := false
			for _, presentItem := range switchList.Items {
				if presentItem.Name == desiredItem {
					found = true
					break
				}
			}
			if !found {
				switchesToAdd = append(switchesToAdd, desiredItem)
			}
		}
	}

	if len(switchesToRemove) > 0 {
		// TODO
		log.Info("Should delete some switches")
	}
	if len(switchesToCheckState) > 0 {
		log.Info("Should check some switches")
	}
	if len(switchesToAdd) > 0 {
		if details, err := r.sdnClient.GetSwitchDetails(ctx, &pb.SwitchNames{
			Names: switchesToAdd,
		}); err != nil {
			log.Error(err, "Failed to fetch switch details")
			return err
		} else {
			programNames := sets.New[string](0)
			for _, switchDetails := range details.Details {
				if err := r.Client.Create(ctx, r.buildIncSwitchCR(switchDetails, req)); err != nil {
					log.Error(err, "Failed to add switch details")
					return err
				}
				programNames.Insert(switchDetails.InstalledProgram)
			}
			if err := r.reconcileP4Programs(ctx, req, programNames); err != nil {
				log.Error(err, "Failed to reconcile p4 programs")
				return err
			}
		}
	}
	return nil
}

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

	if shimConfig.Spec.SdnConfig.SdnType == incv1alpha1.KindaSDN {
		if r.sdnClient == nil {
			if err := r.runSdnClient(shimConfig.Spec.SdnConfig.SdnGrpcAddr); err != nil {
				log.Error(err, "Failed to connect to SDN controller")
				return ctrl.Result{}, err
			}
		}
		topo, changed, err := r.reconcileTopology(ctx, req)
		if err != nil {
			return ctrl.Result{}, err
		}
		if changed {
			log.Info("Updated topology")
		}
		if err := r.reconcileIncSwitches(ctx, req, topo); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		return ctrl.Result{}, errors.New("unsupported SDN type")
	}
	
	log.Info("Reconciled SDNShim")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SDNShimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.SDNShim{}).
		Complete(r)
}
