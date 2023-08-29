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

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var quaysecretenginerolelog = logf.Log.WithName("quaysecretenginerole-resource")

func (r *QuaySecretEngineRole) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-redhatcop-redhat-io-v1alpha1-quaysecretenginerole,mutating=true,failurePolicy=fail,sideEffects=None,groups=redhatcop.redhat.io,resources=quaysecretengineroles,verbs=create,versions=v1alpha1,name=mquaysecretenginerole.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &QuaySecretEngineRole{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *QuaySecretEngineRole) Default() {
	quaysecretenginerolelog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:path=/validate-redhatcop-redhat-io-v1alpha1-quaysecretenginerole,mutating=false,failurePolicy=fail,sideEffects=None,groups=redhatcop.redhat.io,resources=quaysecretengineroles,verbs=create;update,versions=v1alpha1,name=vquaysecretenginerole.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &QuaySecretEngineRole{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *QuaySecretEngineRole) ValidateCreate() (admission.Warnings, error) {
	quaysecretenginerolelog.Info("validate create", "name", r.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *QuaySecretEngineRole) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	quaysecretenginerolelog.Info("validate update", "name", r.Name)

	// the path cannot be updated
	if r.Spec.Path != old.(*QuaySecretEngineRole).Spec.Path {
		return nil, errors.New("spec.path cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *QuaySecretEngineRole) ValidateDelete() (admission.Warnings, error) {
	quaysecretenginerolelog.Info("validate delete", "name", r.Name)

	return nil, nil
}
