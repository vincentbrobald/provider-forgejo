// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	tfprovider "brobald.be/provider-forgejo/internal/tfprovider"
	"brobald.be/provider-forgejo/internal/tfversion"
)

func main() {
	err := providerserver.Serve(context.Background(), tfprovider.New(tfversion.Version),
		providerserver.ServeOpts{Address: "registry.opentofu.org/svalabs/forgejo"})
	if err != nil {
		log.Fatal(err)
	}
}
