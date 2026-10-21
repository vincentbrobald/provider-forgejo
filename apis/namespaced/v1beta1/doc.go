// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

// Package v1beta1 holds the ProviderConfig, ClusterProviderConfig and ProviderConfigUsage kinds of
// the namespaced group forgejo.m.crossplane.brobald.be: how the provider connects to
// Forgejo, and which managed resources use each configuration.
// +kubebuilder:object:generate=true
// +groupName=forgejo.m.crossplane.brobald.be
// +versionName=v1beta1
package v1beta1
