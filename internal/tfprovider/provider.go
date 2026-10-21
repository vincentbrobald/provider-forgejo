// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package tfprovider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	vendored "brobald.be/provider-forgejo/third_party/terraform-provider-forgejo/pkg/provider"
)

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

func (p *withExtras) Resources(ctx context.Context) []func() resource.Resource {
	return append(p.Provider.Resources(ctx), NewTeamRepositoryResource, NewOrganizationOwnerResource)
}
