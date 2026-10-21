// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

// Package v1alpha1 is the API version v1alpha1 of the cluster-scoped group
// forgejo.crossplane.brobald.be. It holds no kinds: the managed resource kinds are in the
// packages of their own groups (organization, repository, user).
// +kubebuilder:object:generate=true
// +groupName=forgejo.crossplane.brobald.be
// +versionName=v1alpha1
package v1alpha1
