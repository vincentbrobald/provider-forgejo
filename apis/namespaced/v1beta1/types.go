// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// A ProviderConfigSpec is the desired state of a ProviderConfig.
type ProviderConfigSpec struct {
	// Credentials required to authenticate to Forgejo: its URL (host) and
	// either a token (api_token) or a user's name and password (username,
	// password; needed for PersonalAccessToken). A Secret holds them either
	// as JSON under the selected key ({"host": "https://...", "api_token":
	// "..."}) or, when the Secret has no such key, as flat keys of these
	// names. A namespaced ProviderConfig's Secret is read from its own
	// namespace.
	Credentials ProviderCredentials `json:"credentials"`
}

// ProviderCredentials says where the provider reads its Forgejo credentials.
type ProviderCredentials struct {
	// Source of the provider credentials: Secret (JSON or flat keys), or
	// JSON from the environment or a file.
	// +kubebuilder:validation:Enum=Secret;Environment;Filesystem
	Source xpv2.CredentialsSource `json:"source"`

	xpv2.CommonCredentialSelectors `json:",inline"`
}

// A ProviderConfigStatus is the observed state of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv2.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true

// A ProviderConfig configures how the provider connects to Forgejo. Managed
// resources of its own namespace select it (kind ProviderConfig); its
// credentials Secret is read from that namespace.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,forgejo}
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// A ProviderConfigList is a list of ProviderConfigs.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true

// A ProviderConfigUsage records that a managed resource uses a
// ProviderConfig or ClusterProviderConfig. A configuration isn't deleted
// while it has usages.
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,forgejo}
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.TypedProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// A ProviderConfigUsageList is a list of ProviderConfigUsages.
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}

// +kubebuilder:object:root=true

// A ClusterProviderConfig configures how the provider connects to Forgejo,
// for managed resources of any namespace (kind ClusterProviderConfig). Its
// credentials Secret is read from the namespace its secretRef names.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="SECRET-NAME",type="string",JSONPath=".spec.credentials.secretRef.name",priority=1
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,forgejo}
type ClusterProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// A ClusterProviderConfigList is a list of ClusterProviderConfigs.
type ClusterProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterProviderConfig `json:"items"`
}
