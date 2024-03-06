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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	incv1alpha1 "operators/api/v1alpha1"
)

// IncSwitchReconciler reconciles a IncSwitch object
type IncSwitchReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const NO_ENTRIES = -1

//+kubebuilder:rbac:groups=inc.kntp.com,resources=incswitches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inc.kntp.com,resources=incswitches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inc.kntp.com,resources=incswitches/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IncSwitch object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *IncSwitchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	incSwitch := &incv1alpha1.IncSwitch{}
	if err := r.Client.Get(ctx, req.NamespacedName, incSwitch); err != nil {
		log.Error(err, "Failed to get incswitch resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	reinstalled, err := r.reinstallProgramIfNeeded(ctx, incSwitch)
	if err != nil {
		log.Error(err, "reinstall")
		return ctrl.Result{}, err
	}

	if reinstalled {
		// we changed status, requeue
		return ctrl.Result{Requeue: true}, nil
	} 

	if incSwitch.Status.InstalledEntriesRevision == NO_ENTRIES {
		if err := r.writeAllMatchActionEntries(ctx, incSwitch); err != nil {
			return ctrl.Result{}, err
		}
		incSwitch.Status.InstalledEntriesRevision = 999 // TODO
		err = r.Status().Update(ctx, incSwitch)
		return ctrl.Result{}, err
	}

	curEntriesRevision := int64(0) // get revision number from table config meta
	if incSwitch.Status.InstalledEntriesRevision != curEntriesRevision {
		entryDiff, err := r.getDifferingEntries(ctx, incSwitch)
		if err != nil {
			return ctrl.Result{}, err
		}
		if len(entryDiff) > 0 {
			if err = r.writeEntries(ctx, incSwitch, entryDiff); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		log.Info("Entries are up to date")
	}

	return ctrl.Result{}, nil
}


func (r *IncSwitchReconciler) reinstallProgramIfNeeded(ctx context.Context, incSwitch *incv1alpha1.IncSwitch) (reinstalled bool, err error) {
	if incSwitch.Spec.ProgramName != incSwitch.Status.InstalledProgram {
		log := log.FromContext(ctx)
		log.Info("Installing program...")
		time.Sleep(time.Second * 2)
		log.Info("Install finished")
		incSwitch.Status.InstalledProgram = incSwitch.Spec.ProgramName
		incSwitch.Status.InstalledEntriesRevision = NO_ENTRIES
		err = r.Status().Update(ctx, incSwitch)
		if err != nil {
			return
		}
		reinstalled = true
		return
	}
	return false, nil
}

func (r *IncSwitchReconciler) writeAllMatchActionEntries(ctx context.Context, incSwitch *incv1alpha1.IncSwitch) error {
	log := log.FromContext(ctx)
	log.Info("Writing all entries")
	return r.writeEntries(ctx, incSwitch, []string{"e1", "e2"})
}

func (r *IncSwitchReconciler) writeEntries(ctx context.Context, incSwitch *incv1alpha1.IncSwitch, entry []string) error {
	return nil
}

func (r *IncSwitchReconciler) getDifferingEntries(ctx context.Context, incSwitch *incv1alpha1.IncSwitch) ([]string, error) {
	return []string{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IncSwitchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&incv1alpha1.IncSwitch{}).
		Complete(r)
}
