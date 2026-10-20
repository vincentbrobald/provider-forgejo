# Add a kind

> How-to · [Documentation](../../README.md#documentation)

Install another Forgejo object as a Crossplane kind.

## From the Terraform provider

Add the resource to `ExternalNameConfigs`
(`config/external_name.go`) with its external name, and its group and kind
in `config/repository.go` (or a new file), then `make generate`. Check
whether a field holds a secret that upstream doesn't mark sensitive, and
whether it has a numeric `id` (Upjet types every `id` as a string: see the
`idAsString` conversion in `config/repository.go`).

## A resource the Terraform provider lacks

Write it in `internal/tfprovider` (Plugin
Framework, using the vendored provider's `*forgejo.Client` from
`Configure`), add it to `Resources` there, then `make schema generate` and
configure it like any other (`forgejo_team_repository` is the example). The
vendored files stay unchanged.
