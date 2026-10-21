// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package features

import (
	xpfeature "github.com/crossplane/crossplane-runtime/v2/pkg/feature"
)

const (
	EnableAlphaExternalSecretStores xpfeature.Flag = "EnableAlphaExternalSecretStores"

	EnableBetaManagementPolicies xpfeature.Flag = xpfeature.EnableBetaManagementPolicies
)
