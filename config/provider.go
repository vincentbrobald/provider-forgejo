// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

package config

import (
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

// Upjet always generates them; the namespaced ones (GetProviderNamespaced)
// are the ones to use with Crossplane v2.
func GetProvider() *ujconfig.Provider {
	return newProvider("forgejo.crossplane.brobald.be")
}

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
