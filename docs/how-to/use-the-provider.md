# Use the provider

> How-to · [Documentation](../../README.md#documentation)

Install the provider in a cluster, give it Forgejo credentials, and manage
objects with it. Every kind: [Kinds](../reference/kinds.md).

## Install and configure

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata: {name: provider-forgejo}
spec:
  package: registry.example.org/providers/forgejo:<VERSION>-<commit>
---
apiVersion: forgejo.m.crossplane.brobald.be/v1beta1
kind: ClusterProviderConfig
metadata: {name: default}
spec:
  credentials:
    source: Secret
    secretRef: {namespace: crossplane-system, name: forgejo-credentials, key: credentials}
```

The credentials key holds JSON: `{"host": "https://forgejo.example", "api_token": "..."}`
(or `username` and `password` instead of the token). When the Secret has no
such key, its flat keys `host`, `api_token` (or `username`, `password`) are
read instead, so that a tool can keep the token in a key of its own. The token's user needs
admin rights on the repositories it manages (a collaborator with `admin`
permission is enough). The provider checks Forgejo's certificate against
the system CAs: add a private CA with a `DeploymentRuntimeConfig` mounting
it and setting `SSL_CERT_DIR`.

## Manage an object

```yaml
apiVersion: repository.forgejo.m.crossplane.brobald.be/v1alpha1
kind: Repository
metadata: {name: demo, namespace: team-a}
spec:
  managementPolicies: [Observe, Create, Update]   # never deleted by Crossplane
  forProvider: {name: demo, private: false, description: "Demo", autoInit: true}
---
apiVersion: repository.forgejo.m.crossplane.brobald.be/v1alpha1
kind: DeployKey
metadata: {name: demo-ci, namespace: team-a}
spec:
  forProvider:
    repositoryIdRef: {name: demo}
    title: ci
    readOnly: true
    key: ssh-ed25519 AAAA... ci
```

A repository with a deploy key; `Delete` is left out of the repository's
`managementPolicies`, so removing the object never deletes the repository.

## Adopt an existing object

Set `crossplane.io/external-name` to its
external name (for a repository, its ID) and start with
`managementPolicies: [Observe]`; compare
`status.atProvider` with what Forgejo has, then add `Create`/`Update`.
Leaving `Delete` out keeps the object in Forgejo when the Kubernetes object
goes. A collaborator needs no annotation: it is found by repository and user.
