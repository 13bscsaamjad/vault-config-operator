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
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	"github.com/redhat-cop/operator-utils/pkg/util"
	utilstemplates "github.com/redhat-cop/operator-utils/pkg/util/templates"
	redhatcopv1alpha1 "github.com/redhat-cop/vault-config-operator/api/v1alpha1"
	vaultutils "github.com/redhat-cop/vault-config-operator/api/v1alpha1/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// VaultSecretReconciler reconciles a VaultSecret object
type VaultSecretReconciler struct {
	util.ReconcilerBase
	Log            logr.Logger
	ControllerName string
}

//+kubebuilder:rbac:groups=redhatcop.redhat.io,resources=vaultsecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=redhatcop.redhat.io,resources=vaultsecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=redhatcop.redhat.io,resources=vaultsecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VaultSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *VaultSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the instance
	instance := &redhatcopv1alpha1.VaultSecret{}
	err := r.GetClient().Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	ctx = context.WithValue(ctx, "kubeClient", r.GetClient())

	err = r.manageReconcileLogic(ctx, instance)
	if err != nil {
		r.Log.Error(err, "unable to complete reconcile logic", "instance", instance)
		return r.ManageError(ctx, instance, err)
	}

	nextUpdateTime := instance.Status.LastVaultSecretUpdate.Add(r.calculateDuration(instance))

	nextTimestamp := metav1.NewTime(nextUpdateTime)
	instance.Status.NextVaultSecretUpdate = &nextTimestamp

	//we reschedule the next reconcile at the time in the future corresponding to
	nextSchedule := time.Until(nextUpdateTime)
	if nextSchedule > 0 {
		return r.ManageSuccessWithRequeue(ctx, instance, nextSchedule)
	} else {
		return r.ManageSuccessWithRequeue(ctx, instance, time.Second)
	}

}

func (r *VaultSecretReconciler) formatK8sSecret(instance *redhatcopv1alpha1.VaultSecret, data interface{}) (*corev1.Secret, error) {

	stringData := make(map[string]string)
	for k, v := range instance.Spec.TemplatizedK8sSecret.StringData {

		tpl, err := template.New("").Funcs(utilstemplates.AdvancedTemplateFuncMap(r.GetRestConfig(), r.Log)).Parse(v)
		if err != nil {
			r.Log.Error(err, "unable to create template", "instance", instance)
			return nil, err
		}

		var b bytes.Buffer
		err = tpl.Execute(&b, data)
		if err != nil {
			r.Log.Error(err, "unable to execute template", "instance", instance)
			return nil, err
		}

		stringData[k] = b.String()
	}

	k8sSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Spec.TemplatizedK8sSecret.Name,
			Namespace:   instance.Namespace,
			Annotations: instance.Spec.TemplatizedK8sSecret.Annotations,
			Labels:      instance.Spec.TemplatizedK8sSecret.Labels,
		},
		StringData: stringData,
		Type:       corev1.SecretType(instance.Spec.TemplatizedK8sSecret.Type),
	}

	//ctrl.SetControllerReference(instance, k8sSecret, r.GetScheme())

	return k8sSecret, nil
}

func (r *VaultSecretReconciler) calculateDuration(instance *redhatcopv1alpha1.VaultSecret) time.Duration {

	// if set, always use refresh period if set
	if instance.Spec.RefreshPeriodOverride != nil {
		return instance.Spec.RefreshPeriodOverride.Duration
	}

	if instance.Status.LastVaultSecretUpdate != nil {
		// use the smallest leaseduration in the VaultDefinitionsStatus array
		var smallestLeaseDurationSeconds int = math.MaxInt64
		for _, defstat := range instance.Status.VaultSecretDefinitionsStatus {
			if defstat.LeaseDuration < smallestLeaseDurationSeconds {
				smallestLeaseDurationSeconds = defstat.LeaseDuration
			}
		}

		percentage := float64(instance.Spec.LeaseDurationRefreshScale) / float64(100)
		scaledSeconds := float64(smallestLeaseDurationSeconds) * percentage
		duration := time.Duration(scaledSeconds) * time.Second
		return duration
	}

	//No release period or durations known
	return instance.Spec.DefaultRefreshPeriod.Duration

}

func (r *VaultSecretReconciler) manageReconcileLogic(ctx context.Context, instance *redhatcopv1alpha1.VaultSecret) error {

	// If the resync period has not elapsed don't reconcile
	if instance.Status.LastVaultSecretUpdate != nil && !instance.Status.LastVaultSecretUpdate.Add(r.calculateDuration(instance)).Before(time.Now()) {
		return nil
	}

	r.Log.V(1).Info("Reconcile VaultSecret", "namespacedName", fmt.Sprintf("%v/%v", instance.Namespace, instance.Name))

	mergedMap := make(map[string]interface{})

	definitionsStatus := make([]redhatcopv1alpha1.VaultSecretDefinitionStatus, len(instance.Spec.VaultSecretDefinitions))

	for idx, vaultSecretDefinition := range instance.Spec.VaultSecretDefinitions {
		vaultClient, err := vaultSecretDefinition.Authentication.GetVaultClient(ctx, instance.Namespace)
		if err != nil {
			r.Log.Error(err, "unable to create vault client", "instance", instance)
			return err
		}

		ctx = context.WithValue(ctx, "vaultClient", vaultClient)
		vaultEndpoint := vaultutils.NewVaultEndpointObj(&vaultSecretDefinition)
		vaultSecret, ok, _ := vaultEndpoint.ReadSecret(ctx)
		if !ok {
			return errors.New("unable to read Vault Secret for " + vaultSecretDefinition.GetPath())
		}

		definitionsStatus[idx] = redhatcopv1alpha1.VaultSecretDefinitionStatus{
			Name:          vaultSecretDefinition.Name,
			LeaseID:       vaultSecret.LeaseID,
			LeaseDuration: vaultSecret.LeaseDuration,
			Renewable:     vaultSecret.Renewable,
		}

		mergedMap[vaultSecretDefinition.Name] = vaultSecret.Data
	}

	k8sSecret, err := r.formatK8sSecret(instance, mergedMap)
	if err != nil {
		r.Log.Error(err, "unable to format k8s secret", "instance", instance)
		return err
	}

	err = r.CreateOrUpdateResource(ctx, instance, instance.GetNamespace(), k8sSecret)
	if err != nil {
		return err
	}

	now := metav1.NewTime(time.Now())
	instance.Status.LastVaultSecretUpdate = &now
	instance.Status.VaultSecretDefinitionsStatus = definitionsStatus

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VaultSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {

	vaultSecretPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newVaultSecret, ok := e.ObjectNew.DeepCopyObject().(*redhatcopv1alpha1.VaultSecret)
			if !ok {
				return false
			}
			oldVaultSecret, ok := e.ObjectOld.DeepCopyObject().(*redhatcopv1alpha1.VaultSecret)
			if !ok {
				return false
			}

			if !reflect.DeepEqual(oldVaultSecret.Spec, newVaultSecret.Spec) {
				return true
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	k8sSecretPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newK8sSecret, ok := e.ObjectNew.DeepCopyObject().(*corev1.Secret)
			if !ok {
				return false
			}
			oldK8sSecret, ok := e.ObjectOld.DeepCopyObject().(*corev1.Secret)
			if !ok {
				return false
			}

			if !reflect.DeepEqual(oldK8sSecret.Data, newK8sSecret.Data) {
				return true
			}

			if !reflect.DeepEqual(oldK8sSecret.Annotations, newK8sSecret.Annotations) {
				return true
			}

			if !reflect.DeepEqual(oldK8sSecret.Labels, newK8sSecret.Labels) {
				return true
			}

			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redhatcopv1alpha1.VaultSecret{}, builder.WithPredicates(vaultSecretPredicate)).
		Owns(&corev1.Secret{}, builder.WithPredicates(k8sSecretPredicate)).
		Complete(r)
}
