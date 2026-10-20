# CI and releases

> Reference · [Documentation](../../README.md#documentation)

`.tekton/build.yaml` (namespace `provider-forgejo` on cluster lab): vet,
tests, generated code current, then `make build package-yaml` and one image
build (kaniko, Containerfile stage `package`): the provider binary plus
`package.yaml` (crossplane.yaml and the CRDs). That image *is* the Crossplane
package: Crossplane reads `package.yaml` from its files when no layer is
annotated as the package (xpkg specification) and runs the same image as the
provider. Trivy scans it (fails on fixable HIGH/CRITICAL); on `main` skopeo
pushes `<registry>/providers/forgejo:<VERSION>-<commit>` and `:latest` to
the site's Harbor (`<registry>`: key `registry` of the Secret below; project
`providers`), as the robot account
`robot$providers+provider-forgejo-ci`, whose name and password are in Secret
`registry-push` of namespace `provider-forgejo` (gitops, SOPS; Crossplane
makes the robot with that password). Then, unless the image's config or a
layer names the site (its domain, for one), the same tags go to Docker Hub,
`docker.io/vincentbrobald/provider-forgejo`: the package the published copies
of the lab use. Its credentials are Secret `dockerhub-push` (keys `username`,
`token`) of the same namespace. Bump `VERSION` for a release worth naming.

Every image carries the licenses of what it contains under `/licenses`
(`make licenses`: go-licenses copies each dependency's license, and the
source of the MPL-2.0 ones; plus this repository's LICENSE, NOTICE and
LICENSES, and the vendored provider's LICENSE), and labels
`org.opencontainers.image.source` (the public repository) and
`org.opencontainers.image.licenses`.

The CI avoids compiling anything twice and frees Go's caches between steps:
the cluster's nodes have small disks, and a step that fills one is evicted.
`make xpkg` (Crossplane CLI) builds the same package the official way, for
local use.
