// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

//go:build generate
// +build generate

// NOTE: See the below link for details on what is happening here.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

// Delete the previous run's output: the CRDs, the generated files (except
// zz_generated.managed*.go), the generated controllers and the examples.
//go:generate rm -rf ../package/crds

//go:generate bash -c "find . -iname 'zz_*' ! -iname 'zz_generated.managed*.go' -delete"
//go:generate bash -c "find . -type d -empty -delete"
//go:generate bash -c "find ../internal/controller -iname 'zz_*' -delete"
//go:generate bash -c "find ../internal/controller -type d -empty -delete"
//go:generate rm -rf ../examples-generated

// Scrape the vendored provider's resource docs into provider-metadata.yaml
// (descriptions and examples).
//go:generate go run github.com/crossplane/upjet/v2/cmd/scraper -n svalabs/forgejo --resource-prefix forgejo -r ../third_party/terraform-provider-forgejo/docs/resources -o ../config/provider-metadata.yaml

// Run Upjet's pipeline (cmd/generator): Go types, controllers and examples
// of both scopes.
//go:generate go run ../cmd/generator/main.go ..

// Generate the deepcopy methods and the CRDs (package/crds).
//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../hack/boilerplate.go.txt paths=./... crd:allowDangerousTypes=true,crdVersions=v1 output:artifacts:config=../package/crds

// Generate the methods of Crossplane's resource interfaces
// (zz_generated.managed*.go and the like).
//go:generate go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets --header-file=../hack/boilerplate.go.txt ./...

// Package apis holds the go:generate steps of make generate, run in this
// directory; its blank imports keep the generator tools in go.mod.
package apis

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen" //nolint:typecheck

	_ "github.com/crossplane/crossplane-tools/cmd/angryjet" //nolint:typecheck
)
