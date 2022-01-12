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
	"context"

	"github.com/hashicorp/go-multierror"
	vaultutils "github.com/redhat-cop/vault-config-operator/api/v1alpha1/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VaultSecretSpec defines the desired state of VaultSecret
type VaultSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// RefreshPeriod if specified, the operator will refresh the secret with the given frequency.
	// +kubebuilder:validation:Optional
	RefreshPeriod *metav1.Duration `json:"refreshPeriod,omitempty"`
	// VaultSecretDefinitions are the secrets in Vault.
	// +kubebuilder:validation:Required
	VaultSecretDefinitions []VaultSecretDefinition `json:"vaultSecretDefinitions,omitempty"`
	// TemplatizedK8sSecret is the formatted K8s Secret created by templating from the Vault KV secrets.
	// +kubebuilder:validation:Required
	TemplatizedK8sSecret TemplatizedK8sSecret `json:"output,omitempty"`
}

// VaultSecretStatus defines the observed state of VaultSecret
//TODO add the TTL/expiration timestamp to the Status field per VaultSecretDefinition
type VaultSecretStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	//LastVaultSecretUpdate the last time when this secret was updated from Vault
	LastVaultSecretUpdate *metav1.Time `json:"lastVaultSecretUpdate,omitempty"`

	//NextVaultSecretUpdate the next time when this secret will be synced with Vault
	NextVaultSecretUpdate *metav1.Time `json:"nextVaultSecretUpdate,omitempty"`

	//VaultSecretDefinitionsStatus information used to determine if the secret should be rereconciled
	VaultSecretDefinitionsStatus []VaultSecretDefinitionStatus `json:"vaultSecretDefinitionsStatus,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

func (vs *VaultSecret) GetConditions() []metav1.Condition {
	return vs.Status.Conditions
}

func (vs *VaultSecret) SetConditions(conditions []metav1.Condition) {
	vs.Status.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VaultSecret is the Schema for the vaultsecrets API
type VaultSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultSecretSpec   `json:"spec,omitempty"`
	Status VaultSecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VaultSecretList contains a list of VaultSecret
type VaultSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VaultSecret{}, &VaultSecretList{})
}

type VaultSecretDefinition struct {
	// Name is an arbitrary, but unique, name for this KV Vault secret and referenced when templating.
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// Authentication is the kube auth configuraiton to be used to execute this request
	// +kubebuilder:validation:Required
	Authentication KubeAuthConfiguration `json:"authentication,omitempty"`
	// Path is the path of the secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:default=kubernetes
	Path Path `json:"path,omitempty"`
}

type VaultSecretDefinitionStatus struct {
	// Name is an arbitrary, but unique, name for this KV Vault secret and referenced when templating.
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// LeaseID is the id of a lease, this denotes the secret is dynamic
	// +kubebuilder:validation:Optional
	LeaseID string `json:"lease_id,omitempty"`
	// LeaseDuration is the time until the secret should be read in again, thus recreating the k8s Secret
	// +kubebuilder:validation:Optional
	LeaseDuration int `json:"lease_duration,omitempty"`
	// Renewable informs if the lease is renewable for the dynamic secret
	// +kubebuilder:validation:Optional
	Renewable bool `json:"renewable,omitempty"`
}

type TemplatizedK8sSecret struct {
	// Name is the K8s Secret name to output to.
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// Type is the K8s Secret type to output to.
	// +kubebuilder:validation:Required
	Type string `json:"type,omitempty"`
	// StringData is the K8s Secret stringData and allows specifying non-binary secret data in string form with go templating support
	// to transform the Vault KV secrets into a formatted K8s Secret.
	// The Sprig template library and Helm functions (like toYaml) are supported.
	// +kubebuilder:validation:Required
	StringData map[string]string `json:"stringData,omitempty"`
	// Labels are labels to add to the final K8s Secret.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are annotations to add to the final K8s Secret.
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

func (vs *VaultSecret) IsValid() (bool, error) {
	err := vs.isValid()
	return err == nil, err
}

func (vs *VaultSecret) isValid() error {
	result := &multierror.Error{}
	return result.ErrorOrNil()
}

var _ vaultutils.VaultObject = &VaultSecretDefinition{}

func (d *VaultSecretDefinition) GetPath() string {
	return string(d.Path)
}
func (d *VaultSecretDefinition) GetPayload() map[string]interface{} {
	return nil
}
func (d *VaultSecretDefinition) IsEquivalentToDesiredState(payload map[string]interface{}) bool {
	return false
}

func (d *VaultSecretDefinition) IsInitialized() bool {
	return true
}

func (r *VaultSecretDefinition) IsValid() (bool, error) {
	return true, nil
}
func (d *VaultSecretDefinition) PrepareInternalValues(context context.Context, object client.Object) error {
	return nil
}
