// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

// Package features names the provider's feature flags.
package features

import (
	xpfeature "github.com/crossplane/crossplane-runtime/v2/pkg/feature"
)

const (
	// EnableAlphaExternalSecretStores enables external secret stores. Nothing
	// in this provider sets it.
	EnableAlphaExternalSecretStores xpfeature.Flag = "EnableAlphaExternalSecretStores"

	// EnableBetaManagementPolicies enables management policies; main sets it
	// unless --enable-management-policies=false.
	EnableBetaManagementPolicies xpfeature.Flag = xpfeature.EnableBetaManagementPolicies
)
