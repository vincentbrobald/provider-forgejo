# provider-forgejo

A Crossplane provider for Forgejo: organizations, teams, repositories and
their settings (deploy keys, webhooks, collaborators, branch protection) as
Kubernetes objects. Generated with
[Upjet](https://github.com/crossplane/upjet) from the Forgejo Terraform
provider ([svalabs/terraform-provider-forgejo](https://github.com/svalabs/terraform-provider-forgejo),
MPL-2.0), whose Go code is **compiled in**: no Terraform or OpenTofu binary
runs, at any time in the cluster.

## Documentation

The documentation follows [Diátaxis](https://diataxis.fr/): how-to guides for
tasks, reference to look things up, explanation to understand. How the
platform uses it: the gitops repository's
[Manage an external object from Git](https://github.com/vincentbrobald/gitops/blob/main/docs/tutorials/manage-an-external-object.md)
and [Crossplane providers](https://github.com/vincentbrobald/gitops/blob/main/docs/reference/crossplane-providers.md).

**How-to guides**
- [Use the provider](docs/how-to/use-the-provider.md): install, credentials, manage and adopt objects
- [Add a kind](docs/how-to/add-a-kind.md): from the Terraform provider, or one it lacks
- [Upgrade the Terraform provider](docs/how-to/upgrade-the-terraform-provider.md)

**Reference**
- [Kinds](docs/reference/kinds.md): every kind, its external name and rules
- [Make targets](docs/reference/make-targets.md): vendor, schema, generate, test, build, package
- [CI and releases](docs/reference/ci-and-releases.md): the pipeline, the image, the registry
- [Repository layout](docs/reference/layout.md): what is hand-written, what is generated

**Explanation**
- [Design decisions](docs/explanation/design-decisions.md): the choices, the alternatives considered, when to revisit them

## Licenses

Our own code: Mozilla Public License 2.0 (LICENSE), © 2026 Vincent Brobald,
the license of the Terraform provider it is built on. Third-party code keeps
its original license, listed file by file in NOTICE:

- Apache-2.0 (`LICENSES/Apache-2.0.txt`), © The Crossplane Authors: the files
  taken from the Upjet provider template, and the generated code (`zz_*.go`).
- MPL-2.0 (its LICENSE): `third_party/terraform-provider-forgejo` (unchanged
  except for import paths, `VENDORED.md`), and the text the generated schema,
  metadata, examples and CRDs take from it.
