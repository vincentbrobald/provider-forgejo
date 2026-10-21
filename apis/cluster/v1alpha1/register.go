// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Group is the API group of the provider's cluster-scoped kinds.
	Group = "forgejo.crossplane.brobald.be"
	// Version is the API version of this package.
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is the group and version of this package.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder adds this package's kinds to a scheme (none); the
	// AddToScheme of apis/cluster includes it.
	SchemeBuilder = runtime.NewSchemeBuilder()
)

// init registers nothing: this package has no kinds.
func init() {}
