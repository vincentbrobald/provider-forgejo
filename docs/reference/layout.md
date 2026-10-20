# Repository layout

> Reference · [Documentation](../../README.md#documentation)

```
third_party/terraform-provider-forgejo/  vendored Terraform provider (hack/vendor-tf-provider.sh)
config/            Upjet configuration: kinds, external names, sensitive fields (hand-written)
  schema.json      the Terraform provider's schema (make schema)
  provider-metadata.yaml  descriptions and examples from its docs (make generate)
internal/tfprovider the Terraform provider run: the vendored one plus our own
                   resources (forgejo_team_repository, forgejo_organization_owner), sharing its client
internal/clients/  ProviderConfig → Terraform provider configuration (from the Upjet template, adapted)
internal/tfversion the vendored version
apis/, internal/controller/, package/crds/, examples-generated/  generated (make generate)
cmd/provider       the provider's main; cmd/generator: Upjet's code generator
hack/tfprovider    serves the provider (internal/tfprovider) to OpenTofu for `make schema` only
hack/boilerplate.go.txt  header of the generated files (the Upjet template's: Apache-2.0)
LICENSE, NOTICE, LICENSES/  MPL-2.0 for our code; NOTICE lists the third-party files and their licenses
.tekton/build.yaml CI (Pipelines-as-Code on cluster lab)
```
