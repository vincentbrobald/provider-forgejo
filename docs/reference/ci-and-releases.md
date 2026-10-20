# CI and releases

> Reference · [Documentation](../../README.md#documentation)

`.tekton/build.yaml` (namespace `provider-forgejo` on cluster lab): vet,
tests, generated code current, then `make build package-yaml` and one image
build (kaniko, Containerfile stage `package`): the provider binary plus
`package.yaml` (crossplane.yaml and the CRDs). That image *is* the Crossplane
package: Crossplane reads `package.yaml` from its files when no layer is
annotated as the package (xpkg specification) and runs the same image as the
provider. Trivy scans it (fails on fixable HIGH/CRITICAL); on `main` skopeo
pushes `registry.example.org/providers/forgejo:<VERSION>-<commit>`
and `:latest` (Harbor, project `providers`), as the robot account
`robot$providers+provider-forgejo-ci`, whose name and password are in Secret
`registry-push` of namespace `provider-forgejo` (gitops, SOPS; Crossplane
makes the robot with that password). Bump `VERSION` for a release worth naming.

The CI avoids compiling anything twice and frees Go's caches between steps:
the cluster's nodes have small disks, and a step that fills one is evicted.
`make xpkg` (Crossplane CLI) builds the same package the official way, for
local use.
