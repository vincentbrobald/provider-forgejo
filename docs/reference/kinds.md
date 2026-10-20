# Kinds

> Reference · [Documentation](../../README.md#documentation)

Every kind the provider installs, its external name and its rules. To use
them: [Use the provider](../how-to/use-the-provider.md).

Namespaced (Crossplane v2), groups `repository.forgejo.m.crossplane.brobald.be`,
`organization.forgejo.m.crossplane.brobald.be` and `user.forgejo.m.crossplane.brobald.be`,
version `v1alpha1`.
Upjet also generates cluster-scoped twins (without `.m`); use the namespaced
ones.

| Kind | Terraform resource | External name | Notes |
|---|---|---|---|
| `DeployKey` | `forgejo_deploy_key` | `<repository_id>/<key_id>` | `key` is the public key |
| `Webhook` | `forgejo_repository_webhook` | `<repository_id>/<webhook_id>` | `config` (url, content_type, **secret**) comes from a Secret: `configSecretRef` (required, though the CRD doesn't say so). List `events` as Forgejo stores them: it expands `pull_request` into all `pull_request_*` events, and a shorter list differs forever |
| `Collaborator` | `forgejo_collaborator` | `<repository_id>/<user>` | |
| `BranchProtection` | `forgejo_branch_protection` | `<repository_id>/<branch_name>` | |
| `Organization` | `forgejo_organization` | its name | the token's user becomes its owner |
| `Team` | `forgejo_team` | `<id>` (numeric) | `organization` or `organizationRef`; permission, units, `canCreateOrgRepo` |
| `TeamMember` | `forgejo_team_member` | `<team_id>/<user>` | `teamId` or `teamIdRef`; the user must exist (Keycloak users appear after their first login) |
| `OrganizationOwner` | `forgejo_organization_owner` (ours: `internal/tfprovider`) | `<organization>/<user>` | an owner of an organization: a member of its team Owners, found by name (no team ID to look up once the organization exists); `organization`, `user` (who must exist). Forgejo refuses to remove the last owner |
| `TeamRepository` | `forgejo_team_repository` (ours: `internal/tfprovider`) | `<team_id>/<repository>` | gives an organization's repository to one of its teams (the team's permission and units apply): `teamId` or `teamIdRef`, `organization`, `repository` or `repositoryRef` (a Repository's name) |
| `User` | `forgejo_user` | `<id>` (numeric) | a local user: `login`, `email`, `passwordSecretRef`; set `mustChangePassword: false` for a bot. Needs a provider user that is a **site admin** with a token of scope `write:admin`. Deleting it deletes the user (or deactivates it with `deactivateOnDestroy: true`) |
| `PersonalAccessToken` | `forgejo_personal_access_token` | `<user>/<name>` | a token of `user` (`scopes`, `name`; a change of either is a new token: rename it to rotate), published as `attribute.token` of the connection Secret (`writeConnectionSecretToRef`). Forgejo makes tokens only for the user who signs in with a password: its ProviderConfig holds that user's `username` and `password`, not a token |
| `Repository` | `forgejo_repository` | `<id>` (Forgejo's numeric ID) | public or private (`private`), description, merge styles, wiki, issues... **Deleting it deletes the repository** (or archives it with `archiveOnDestroy: true`): leave `Delete` out of `managementPolicies` for one that matters |

The per-repository kinds take `repositoryId` (the repository's number:
`GET /api/v1/repos/<owner>/<name>`, field `id`; it never changes), or
`repositoryIdRef: {name: <Repository object>}` to use a `Repository`'s ID.

A `Repository` without `owner` belongs to the token's user; `owner` (or
`ownerRef` to an Organization) puts it in an organization, which needs a
token with `write:organization` whose user may create repositories there. Before its creation it has no external name: Crossplane asks for
ID 0, which Forgejo reports as missing, and creates it.

**Objects Forgejo numbers (`Repository`, `Team`, `User`) with `managementPolicies`:**
include `LateInitialize` (e.g. `[Observe, Create, Update, LateInitialize]`)
unless the external name is set by hand. Crossplane saves the ID it learns
after the creation only through a late-initialization update: without it the
external name stays empty, references to the object never resolve, and after
a provider restart Crossplane would create it again.

A missing object (Forgejo answers 404) reads as "does not exist", so
Crossplane creates it; a missing *repository* stays an error.

**Credentials in errors:** Forgejo repeats a rejected password in its
error (it tries it as a token too), and such errors land in the managed
resources' status. The provider removes the configured password and token,
and Forgejo's echoes of them, from the errors of its configuration
(`internal/tfprovider/redact.go`).
