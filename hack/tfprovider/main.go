// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

// Command tfprovider serves the Forgejo Terraform provider (the vendored one
// with this provider's own resources) as a Terraform plugin; make schema runs
// it under OpenTofu to write config/schema.json.
package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	tfprovider "brobald.be/provider-forgejo/internal/tfprovider"
	"brobald.be/provider-forgejo/internal/tfversion"
)

// main serves the provider under the address
// registry.opentofu.org/svalabs/forgejo until OpenTofu stops it.
func main() {
	err := providerserver.Serve(context.Background(), tfprovider.New(tfversion.Version),
		providerserver.ServeOpts{Address: "registry.opentofu.org/svalabs/forgejo"})
	if err != nil {
		log.Fatal(err)
	}
}
