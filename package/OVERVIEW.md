# provider-forgejo

A [Crossplane](https://www.crossplane.io/) provider for
[Forgejo](https://forgejo.org/): organizations, teams, repositories and their
settings (deploy keys, webhooks, collaborators, branch protection), users and
their tokens, as Kubernetes objects.

It is generated with [Upjet](https://github.com/crossplane/upjet) from the
Forgejo Terraform provider
([svalabs/terraform-provider-forgejo](https://github.com/svalabs/terraform-provider-forgejo)),
whose Go code is **compiled in**: no Terraform or OpenTofu binary runs, at
any time.

## What this image is

The Crossplane package itself: Crossplane reads the package metadata and
CRDs (`/package.yaml`) from it and runs the same image as the provider. The
provider binary is `/usr/local/bin/provider`, on distroless static, as a
non-root user. The licenses of everything in it are under `/licenses`.

## Install

Crossplane v2:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata: {name: forgejo}
spec:
  package: docker.io/vincentbrobald/provider-forgejo:<tag>
---
apiVersion: forgejo.m.crossplane.brobald.be/v1beta1
kind: ClusterProviderConfig
metadata: {name: default}
spec:
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: forgejo-credentials, key: credentials}
```

The Secret's key holds `{"host": "https://forgejo.example.org", "api_token": "..."}`.
The provider supports safe-start: activate its kinds with a
ManagedResourceActivationPolicy. Details:
[Use the provider](https://github.com/vincentbrobald/provider-forgejo/blob/main/docs/how-to/use-the-provider.md).

## Kinds

Each kind exists namespaced (`*.forgejo.m.crossplane.brobald.be`) and
cluster-scoped (`*.forgejo.crossplane.brobald.be`):

| Group | Kinds |
|---|---|
| `repository.` | `Repository`, `DeployKey`, `Webhook`, `Collaborator`, `BranchProtection`, `TeamRepository` |
| `organization.` | `Organization`, `OrganizationOwner`, `Team`, `TeamMember` |
| `user.` | `User`, `PersonalAccessToken` |

External names, required fields and caveats:
[Kinds](https://github.com/vincentbrobald/provider-forgejo/blob/main/docs/reference/kinds.md).

## Tags

- `v<version>-<commit>`: one per commit of the main branch (e.g. `v0.7.0-836701d`);
  pin one of these.
- `latest`: the newest of them.

## Source and licenses

Source: [github.com/vincentbrobald/provider-forgejo](https://github.com/vincentbrobald/provider-forgejo).
Our own code is under the Mozilla Public License 2.0; the files from
Crossplane's Upjet provider template and the generated code keep the Apache
License 2.0, the vendored Terraform provider the MPL-2.0 (the repository's
NOTICE lists every file). `/licenses` in the image holds these and every
dependency's license.
