/*
Copyright 2021.

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

	"github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/api/v1alpha4"
	"gitlab.aws.dev/ce-pike/merida/cluster-api-provider-capc/pkg/cloud"
)

// CloudStackMachineReconciler reconciles a CloudStackMachine object
type CloudStackMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	CS     *cloudstack.CloudStackClient
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=cloudstackmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CloudStackMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, retErr error) {
	log := log.FromContext(ctx)

	// Fetch the CloudStackMachine.
	csMachine := &infrav1.CloudStackMachine{}
	if retErr = r.Client.Get(ctx, req.NamespacedName, csMachine); retErr != nil {
		if client.IgnoreNotFound(retErr) == nil {
			log.Info("Machine not found.")
		}
		return ctrl.Result{}, client.IgnoreNotFound(retErr)
	}

	// Setup patcher. This ensures modifications to the csMachine copy fetched above are patched into the origin.
	if patchHelper, retErr := patch.NewHelper(csMachine, r.Client); retErr != nil {
		return ctrl.Result{}, retErr
	} else {
		defer func() { // If there was no error on return, but the patch fails, set the error accordingly.
			if err := patchHelper.Patch(ctx, csMachine); retErr == nil && err != nil {
				retErr = err
			}
		}()
	}

	// Fetch the CAPI Machine.
	machine, retErr := util.GetOwnerMachine(ctx, r.Client, csMachine.ObjectMeta)
	if retErr != nil {
		return ctrl.Result{}, retErr
	} else if machine == nil {
		log.Info("Waiting for CAPI cluster controller to set owner reference on CloudStack machine.")
		return ctrl.Result{}, nil
	}

	// Fetch the CAPI Cluster.
	cluster, retErr := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if retErr != nil {
		log.Info("Machine is missing cluster label or cluster does not exist.")
		return ctrl.Result{}, nil
	}

	// Fetch the CloudStack cluster associated with this machine.
	csCluster := &infrav1.CloudStackCluster{}
	if retErr := r.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: csMachine.Namespace,
			Name:      cluster.Spec.InfrastructureRef.Name},
		csCluster); retErr != nil {
		return ctrl.Result{}, client.IgnoreNotFound(retErr)
	} else if csCluster.Status.ZoneID == "" {
		log.Info("Cluster not found. Likely not ready.")
		return ctrl.Result{}, nil
	}

	// Delete VM instance if deletion timestamp present.
	if !csMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(log, csMachine)
	}

	// Otherwise reconcile a VM instance.
	return r.reconcile(log, csCluster, csMachine)
}

// Actually reconcile/Create a VM instance.
func (r *CloudStackMachineReconciler) reconcile(
	log logr.Logger,
	csCluster *infrav1.CloudStackCluster,
	csMachine *infrav1.CloudStackMachine) (ctrl.Result, error) {

	// Create machine (or Fetch if present). Will set ready to true.
	err := cloud.CreateVMInstance(r.CS, csMachine, csCluster)
	if err == nil {
		log.Info("Machine Created", "instanceStatus", csMachine.Status, "instanceSpec", csMachine.Spec)

		// Prevent premature deletion of the csMachine construct from CAPI.
		controllerutil.AddFinalizer(csMachine, infrav1.MachineFinalizer)
	}

	return ctrl.Result{}, err
}

// Reconcile/Destroy a deleted VM instance.
func (r *CloudStackMachineReconciler) reconcileDelete(
	log logr.Logger,
	csMachine *infrav1.CloudStackMachine) (retRes ctrl.Result, retErr error) {

	defer func() { // Don't remove finalizer if VM destroy returned an error.
		if retErr == nil {
			controllerutil.RemoveFinalizer(csMachine, infrav1.MachineFinalizer)
		}
	}()

	log.Info("Deleting instance", "instance-id", *csMachine.Spec.InstanceID)
	return ctrl.Result{}, cloud.DestroyVMInstance(r.CS, csMachine)
}

// Called in main, this registers the machine reconciler to the CAPI controller manager.
func (r *CloudStackMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&infrav1.CloudStackMachine{}).Complete(r)
}