# Upgrade the Terraform provider

> How-to · [Documentation](../../README.md#documentation)

Move to a new release of the vendored Forgejo Terraform provider.

Set `VERSION` and `SHA256` in
`hack/vendor-tf-provider.sh` and `internal/tfversion`, then `make vendor
schema generate`, review the CRD diff (a field gone or changed type breaks
existing objects) and the external-name assumptions (read errors' wording,
see `config/external_name.go`), `make test`.
