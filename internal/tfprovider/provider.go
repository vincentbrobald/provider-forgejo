// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

// Package tfprovider is the Terraform provider compiled into the Crossplane
// provider: the vendored Forgejo provider, with resources of our own and
// credentials removed from its configuration errors.
package tfprovider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	vendored "brobald.be/provider-forgejo/third_party/terraform-provider-forgejo/pkg/provider"
)

// New returns the constructor of the provider, reporting version as its own.
// Each call of the constructor makes a new instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &withExtras{Provider: vendored.New(version)()}
	}
}

// withExtras forwards everything to the vendored provider (which implements
// only the base provider.Provider interface) except Resources, which adds
// ours, and Configure, which removes credentials from its errors
// (redact.go).
type withExtras struct {
	provider.Provider
}

// Resources returns the vendored provider's resources followed by
// forgejo_team_repository and forgejo_organization_owner.
func (p *withExtras) Resources(ctx context.Context) []func() resource.Resource {
	return append(p.Provider.Resources(ctx), NewTeamRepositoryResource, NewOrganizationOwnerResource)
}
