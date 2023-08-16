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

package vaultresourcecontroller

import (
	"context"
	"time"

	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/redhat-cop/operator-utils/pkg/util/apis" // TODO this dependency should be removed
	vaultutils "github.com/redhat-cop/vault-config-operator/api/v1alpha1/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const ReconcileSuccessful = "ReconcileSuccessful"
const ReconcileSuccessfulReason = "LastReconcileCycleSucceded"
const ReconcileFailed = "ReconcileFailed"
const ReconcileFailedReason = "LastReconcileCycleFailed"

func ManageOutcomeWithRequeue(context context.Context, r util.ReconcilerBase, obj client.Object, issue error, requeueAfter time.Duration) (reconcile.Result, error) {
	log := log.FromContext(context)
	conditionsAware := (obj).(apis.ConditionsAware)
	var condition metav1.Condition
	if issue == nil {
		condition = metav1.Condition{
			Type:               ReconcileSuccessful,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
			Reason:             ReconcileSuccessfulReason,
			Status:             metav1.ConditionTrue,
		}
	} else {
		r.GetRecorder().Event(obj, "Warning", "ProcessingError", issue.Error())
		condition = metav1.Condition{
			Type:               ReconcileFailed,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
			Message:            issue.Error(),
			Reason:             ReconcileFailedReason,
			Status:             metav1.ConditionFalse,
		}
	}
	conditionsAware.SetConditions(apis.AddOrReplaceCondition(condition, conditionsAware.GetConditions()))
	err := r.GetClient().Status().Update(context, obj)
	if err != nil {
		log.Error(err, "unable to update status")
		return reconcile.Result{}, err
	}
	if issue == nil && !controllerutil.ContainsFinalizer(obj, vaultutils.GetFinalizer(obj)) {
		controllerutil.AddFinalizer(obj, vaultutils.GetFinalizer(obj))
		// BEWARE: this call *mutates* the object in memory with Kube's response, there *must be invoked last*
		err := r.GetClient().Update(context, obj)
		if err != nil {
			log.Error(err, "unable to add reconciler")
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{RequeueAfter: requeueAfter}, issue
}

func ManageOutcome(context context.Context, r util.ReconcilerBase, obj client.Object, issue error) (reconcile.Result, error) {
	return ManageOutcomeWithRequeue(context, r, obj, issue, 0)
}

func isValid(obj client.Object) (bool, error) {
	return obj.(vaultutils.VaultObject).IsValid()
}

// ResourceGenerationChangedPredicate this predicate will fire an update event when the spec of a resource is changed (controller by ResourceGeneration), or when the finalizers are changed
type ResourceGenerationChangedPredicate struct {
	predicate.Funcs
}

// Update implements default UpdateEvent filter for validating resource version change
func (ResourceGenerationChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		return false
	}
	if e.ObjectNew == nil {
		return false
	}
	if e.ObjectNew.GetGeneration() == e.ObjectOld.GetGeneration() {
		return false
	}
	return true
}
