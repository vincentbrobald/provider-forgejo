// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

// Package tfprovider is the Terraform provider that provider-forgejo runs: the
// vendored svalabs provider (third_party/terraform-provider-forgejo, files
// unchanged) plus resources of our own that it lacks, sharing its
// configuration and Forgejo client.
package tfprovider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	vendored "brobald.be/provider-forgejo/third_party/terraform-provider-forgejo/pkg/provider"
)

// New returns the provider factory: the vendored provider's own, with the
// extra resources appended.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &withExtras{Provider: vendored.New(version)()}
	}
}

// withExtras forwards everything to the vendored provider (which implements
// only the base provider.Provider interface) except Resources.
type withExtras struct {
	provider.Provider
}

// Resources: the vendored provider's, then ours.
func (p *withExtras) Resources(ctx context.Context) []func() resource.Resource {
	return append(p.Provider.Resources(ctx), NewTeamRepositoryResource, NewOrganizationOwnerResource)
}
