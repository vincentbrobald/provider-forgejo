# Design decisions

> Explanation · [Documentation](../../README.md#documentation)

The choices provider-forgejo is built on, what else was considered, and when
to look at them again. Why Forgejo's objects are managed by Crossplane at all:
the gitops repository's [Design decisions](https://github.com/vincentbrobald/gitops/blob/main/docs/explanation/design-decisions.md).

### API groups under the owner's public domain

**Decision.** The kinds live in `forgejo.crossplane.brobald.be` and `forgejo.m.crossplane.brobald.be` (with sub-groups `organization.`, `repository.`, `user.`), and the Go module is
`brobald.be/provider-forgejo`.
**Why.** A group must be a domain its owner controls; the site's internal
domain was used at first, and it must not appear in what is published
(a sanitized copy of the lab is shared publicly). `brobald.be` is public and
the owner's.
The package moved with it, to `registry.example.org/providers/forgejo`:
Crossplane refuses two packages from the same source, and the old and new
versions run side by side during the migration.
**Cost.** Renaming a group creates new kinds: the objects of the old group
are migrated to the new one on the cluster (gitops).
**Revisit if** the provider moves to a shared home (e.g. crossplane-contrib),
whose group it would then take.

### Upjet, with the Terraform provider compiled in

**Decision.** The provider is generated with Upjet from the Forgejo Terraform
provider (svalabs, MPL-2.0), whose Go code is compiled into the provider
binary (Upjet's no-fork mode). No Terraform or OpenTofu runs in the cluster.
**Why.** No Crossplane provider for Forgejo existed; the Terraform provider
already models Forgejo's API and is maintained. Compiling it in keeps the
platform free of a Terraform binary and its state, as the platform prefers
open-source components without extra tooling.
**Alternatives.** provider-terraform (runs Terraform or OpenTofu with state in
the cluster); a provider written by hand (every resource's API calls,
diffing and import ourselves).
**Revisit if** Forgejo gains an official Crossplane provider.

### The Terraform provider vendored, unchanged

**Decision.** Its source is vendored into `third_party/`, unchanged except for
import paths (`hack/vendor-tf-provider.sh` moves `internal/` to `pkg/`).
**Why.** Go doesn't allow importing another module's `internal/` packages,
which Upjet's no-fork mode needs. Leaving the files otherwise unchanged
keeps upgrades a re-vendor.

### Missing resources as our own Terraform resources

**Decision.** `forgejo_team_repository` and `forgejo_organization_owner` are
written in `internal/tfprovider` (Plugin Framework, sharing the vendored
provider's client) and served next to the vendored ones.
**Why.** Upjet then generates their kinds like any other; the vendored code
stays untouched.

### OpenTofu only to dump the schema

**Decision.** `make schema` serves the provider to a pinned, checksum-verified
OpenTofu once, to write `config/schema.json`.
**Why.** Upjet reads the schema in Terraform's JSON form; nothing else needs
OpenTofu.

### Namespaced kinds (Crossplane v2)

**Decision.** The namespaced kinds (`*.m.crossplane.brobald.be`) are the
ones to use; the cluster-scoped twins Upjet also generates are not.
**Why.** Objects live with what uses them (a team's namespace), and
admission policies can bound them per namespace.

### Numeric IDs as external names, with LateInitialize

**Decision.** Objects Forgejo numbers (Repository, Team, User) use the ID as
external name; users of them include `LateInitialize`.
**Why.** Crossplane records the ID learned after creation only through a
late-initialization update; without it an object would be created again
after a restart.

### Secrets where upstream doesn't mark them

**Decision.** A webhook's `config` (holding its secret) is forced sensitive
and comes from a Secret; Forgejo's echoes of rejected passwords are stripped
from errors (`internal/tfprovider/redact.go`).
**Why.** Credentials must never land in a managed resource's spec or status.

### The image is the package

**Decision.** One kaniko-built image holds the provider binary and
`package.yaml`; Crossplane reads the package from the image's files.
**Why.** One build and one push in CI, on nodes with small disks; the
Crossplane CLI (`make xpkg`) builds the official form for local use.

### Credentials as flat keys accepted

**Decision.** The ProviderConfig's Secret may hold JSON under one key, or flat
keys (`host`, `api_token`, or `username`, `password`).
**Why.** Tools that keep the token in a key of their own (SOPS files, the
infra playbooks) don't need to compose JSON.
