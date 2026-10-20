// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

// Package config is the Upjet configuration of provider-forgejo: which
// resources of the Forgejo Terraform provider (svalabs, vendored in
// third_party/) become Crossplane kinds, and how each is identified.
//
// The Terraform provider's Go code runs in the provider's own process
// (Terraform Plugin Framework, "no-fork"): no Terraform or OpenTofu binary is
// involved at runtime. schema.json (its schema) and provider-metadata.yaml
// (descriptions and examples from its docs) are inputs of `make generate`.
package config

import (
	// schema.json and provider-metadata.yaml are embedded.
	_ "embed"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"

	tfprovider "brobald.be/provider-forgejo/internal/tfprovider"
	"brobald.be/provider-forgejo/internal/tfversion"
)

const (
	resourcePrefix = "forgejo"
	modulePath     = "brobald.be/provider-forgejo"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// GetProvider returns the configuration of the cluster-scoped kinds
// (forgejo.crossplane.brobald.be). Upjet always generates them; the
// namespaced ones (GetProviderNamespaced) are the ones to use with
// Crossplane v2.
func GetProvider() *ujconfig.Provider {
	return newProvider("forgejo.crossplane.brobald.be")
}

// GetProviderNamespaced returns the configuration of the namespaced kinds
// (forgejo.m.crossplane.brobald.be).
func GetProviderNamespaced() *ujconfig.Provider {
	return newProvider("forgejo.m.crossplane.brobald.be",
		ujconfig.WithExampleManifestConfiguration(ujconfig.ExampleManifestConfiguration{
			ManagedResourceNamespace: "crossplane-system",
		}))
}

func newProvider(rootGroup string, opts ...ujconfig.ProviderOption) *ujconfig.Provider {
	opts = append([]ujconfig.ProviderOption{
		ujconfig.WithRootGroup(rootGroup),
		ujconfig.WithShortName("forgejo"),
		// Only Plugin Framework resources: the provider has no SDKv2 ones,
		// and nothing goes through a Terraform CLI.
		ujconfig.WithIncludeList([]string{}),
		ujconfig.WithTerraformPluginSDKIncludeList([]string{}),
		ujconfig.WithTerraformPluginFrameworkIncludeList(ExternalNameConfigured()),
		ujconfig.WithTerraformPluginFrameworkProvider(tfprovider.New(tfversion.Version)()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(ExternalNameConfigurations()),
	}, opts...)
	pc := ujconfig.NewProvider([]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata), opts...)
	configureRepositoryKinds(pc)
	configureOrganizationKinds(pc)
	configureUserKinds(pc)
	pc.ConfigureResources()
	return pc
}
