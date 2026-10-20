# Make targets

> Reference · [Documentation](../../README.md#documentation)

Go 1.26 (or `make podman-<target>` for the pinned Go image):

| Command | What |
|---|---|
| `make vendor` | re-vendor the Terraform provider (version and checksum in the script) |
| `make schema` | `config/schema.json`, asking the vendored provider through OpenTofu (pinned, checksum-verified) |
| `make generate` | CRDs, Go types, controllers, examples (Upjet, controller-gen, angryjet) |
| `make test` / `make build` | unit tests / `bin/provider` |
| `make licenses` | `bin/licenses`: the licenses of everything in `bin/provider` (go-licenses v2.0.1; the source of MPL-2.0 modules too) and this repository's, for the images' `/licenses` |
| `make package-yaml` | `bin/package.yaml`: `crossplane.yaml` and the CRDs, the package's contents |
| `make image` | the runtime image (podman, distroless static, non-root) as `.build/image.tar` |
| `make xpkg` | the Crossplane package `.build/provider-forgejo.xpkg` (Crossplane CLI v2.5.0, built from source) |
