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
	Group   = "forgejo.crossplane.brobald.be"
	Version = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	SchemeBuilder = runtime.NewSchemeBuilder()
)

func init() {}
