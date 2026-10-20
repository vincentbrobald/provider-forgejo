// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"context"
	"math/big"
	"testing"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestRepoScopedNumericKey(t *testing.T) {
	e := ExternalNameConfigs["forgejo_deploy_key"]

	params := map[string]any{"title": "argocd"}
	e.SetIdentifierArgumentFn(params, "12/345")
	if params["repository_id"] != int64(12) || params["key_id"] != int64(345) {
		t.Fatalf("identifiers not set from the external name: %v", params)
	}

	// Number types the state conversion may produce.
	for _, state := range []map[string]any{
		{"repository_id": float64(12), "key_id": float64(345)},
		{"repository_id": int64(12), "key_id": big.NewFloat(345)},
	} {
		got, err := e.GetExternalNameFn(state)
		if err != nil || got != "12/345" {
			t.Errorf("GetExternalNameFn(%v) = %q, %v; want 12/345", state, got, err)
		}
	}

	if _, err := e.GetExternalNameFn(map[string]any{"repository_id": float64(12)}); err == nil {
		t.Error("a state without key_id must not give an external name")
	}
}

func TestRepoScopedStringKey(t *testing.T) {
	e := ExternalNameConfigs["forgejo_branch_protection"]

	params := map[string]any{}
	e.SetIdentifierArgumentFn(params, "12/release/v1")
	if params["repository_id"] != int64(12) || params["branch_name"] != "release/v1" {
		t.Fatalf("a branch name with slashes must survive: %v", params)
	}

	got, err := e.GetExternalNameFn(map[string]any{"repository_id": float64(12), "branch_name": "main"})
	if err != nil || got != "12/main" {
		t.Errorf("got %q, %v; want 12/main", got, err)
	}
}

func TestRepoScopedIgnoresEmptyExternalName(t *testing.T) {
	params := map[string]any{"user": "pac-bot", "repository_id": int64(7)}
	ExternalNameConfigs["forgejo_collaborator"].SetIdentifierArgumentFn(params, "")
	if params["user"] != "pac-bot" || params["repository_id"] != int64(7) {
		t.Fatalf("an empty external name must leave the spec's values: %v", params)
	}
}

func TestNotFoundDiagnostic(t *testing.T) {
	isNotFound := ExternalNameConfigs["forgejo_deploy_key"].IsNotFoundDiagnosticFn
	notFound := &tfprotov6.Diagnostic{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to read deploy key", Detail: "Deploy key with user a, repo b and ID 3 not found: 404"}
	other := &tfprotov6.Diagnostic{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to read deploy key", Detail: "Unknown error with nil response: dial tcp: refused"}
	repoMissing := &tfprotov6.Diagnostic{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to get repository by id", Detail: "Repository with ID 3 not found: 404"}

	if !isNotFound([]*tfprotov6.Diagnostic{notFound}) {
		t.Error("a missing deploy key must read as not found")
	}
	if isNotFound([]*tfprotov6.Diagnostic{other}) {
		t.Error("a connection error must stay an error")
	}
	if isNotFound([]*tfprotov6.Diagnostic{repoMissing}) {
		t.Error("a missing repository must stay an error, not lead to a creation")
	}
}

func TestEveryConfiguredResourceExists(t *testing.T) {
	pc := GetProviderNamespaced()
	for name := range ExternalNameConfigs {
		r, ok := pc.Resources[name]
		if !ok {
			t.Errorf("%s: not generated", name)
			continue
		}
		want := "repository"
		switch name {
		case "forgejo_organization", "forgejo_team", "forgejo_team_member", "forgejo_organization_owner":
			want = "organization"
		case "forgejo_user", "forgejo_personal_access_token":
			want = "user"
		}
		if r.ShortGroup != want {
			t.Errorf("%s: group %q, want %s", name, r.ShortGroup, want)
		}
	}
	cfg := pc.Resources["forgejo_repository_webhook"].TerraformResource.Schema["config"]
	if !cfg.Sensitive {
		t.Error("the webhook's config (it holds the secret) must be sensitive")
	}
	if cfg.Required {
		t.Error("the webhook's config must be optional for Upjet (else initProvider.configSecretRef is required)")
	}
}

func TestRepositoryExternalName(t *testing.T) {
	e := ExternalNameConfigs["forgejo_repository"]

	// No external name yet: ID 0, which Forgejo reports as not found, so
	// Crossplane creates the repository.
	if id, err := e.GetIDFn(context.Background(), "", nil, nil); err != nil || id != "0" {
		t.Fatalf("GetIDFn(\"\") = %q, %v; want 0", id, err)
	}
	if id, err := e.GetIDFn(context.Background(), "12", nil, nil); err != nil || id != "12" {
		t.Fatalf("GetIDFn(12) = %q, %v; want 12", id, err)
	}
	if got, err := e.GetExternalNameFn(map[string]any{"id": float64(12), "name": "x"}); err != nil || got != "12" {
		t.Fatalf("GetExternalNameFn = %q, %v; want 12", got, err)
	}
	if _, err := e.GetExternalNameFn(map[string]any{"id": "0", "name": "x"}); err == nil {
		t.Fatal("the stand-in ID 0 must never become an external name")
	}
	if got := e.TFPluginFrameworkOptions.ComputedIdentifierAttributes; len(got) != 1 || got[0] != "id" {
		t.Fatalf("id must stay out of the configuration: %v", got)
	}
	repoMissing := &tfprotov6.Diagnostic{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to read repository", Detail: "Repository with ID 0 not found: 404 Not Found"}
	if !e.IsNotFoundDiagnosticFn([]*tfprotov6.Diagnostic{repoMissing}) {
		t.Error("a missing repository must read as not found for the Repository kind")
	}
}

func TestRepositoryReferences(t *testing.T) {
	pc := GetProviderNamespaced()
	for _, name := range []string{"forgejo_deploy_key", "forgejo_repository_webhook", "forgejo_collaborator", "forgejo_branch_protection"} {
		if ref, ok := pc.Resources[name].References["repository_id"]; !ok || ref.TerraformName != "forgejo_repository" {
			t.Errorf("%s: repository_id must reference forgejo_repository", name)
		}
	}
}

func TestRepositoryIDAsString(t *testing.T) {
	r := GetProviderNamespaced().Resources["forgejo_repository"]
	got, err := r.ApplyTFConversions(map[string]any{"id": float64(12), "name": "x"}, ujconfig.FromTerraform)
	if err != nil || got["id"] != "12" {
		t.Fatalf("state id 12 must reach the status as \"12\": %v, %v", got["id"], err)
	}
	got, err = r.ApplyTFConversions(map[string]any{"id": "12"}, ujconfig.ToTerraform)
	if err != nil || got["id"] != "12" {
		t.Fatalf("towards Terraform the id stays as is: %v, %v", got["id"], err)
	}
}

func TestOrganizationExternalName(t *testing.T) {
	e := ExternalNameConfigs["forgejo_organization"]
	params := map[string]any{}
	e.SetIdentifierArgumentFn(params, "acme")
	if params["name"] != "acme" {
		t.Fatalf("the external name is the organization's name: %v", params)
	}
	if got, err := e.GetExternalNameFn(map[string]any{"id": float64(4), "name": "acme"}); err != nil || got != "acme" {
		t.Fatalf("GetExternalNameFn = %q, %v", got, err)
	}
	// The numeric id is unused for reading but must parse as a number.
	if id, _ := e.GetIDFn(context.Background(), "acme", nil, nil); id != "0" {
		t.Fatalf("GetIDFn = %q, want the stand-in 0", id)
	}
	missing := &tfprotov6.Diagnostic{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to read organization", Detail: "Organization with name 'acme' not found: 404"}
	if !e.IsNotFoundDiagnosticFn([]*tfprotov6.Diagnostic{missing}) {
		t.Error("a missing organization must read as not found")
	}
}

func TestTeamAndMemberExternalNames(t *testing.T) {
	team := ExternalNameConfigs["forgejo_team"]
	if id, _ := team.GetIDFn(context.Background(), "", nil, nil); id != "0" {
		t.Fatalf("a team before its creation: %q", id)
	}
	if got, err := team.GetExternalNameFn(map[string]any{"id": float64(7)}); err != nil || got != "7" {
		t.Fatalf("team external name %q, %v", got, err)
	}
	m := ExternalNameConfigs["forgejo_team_member"]
	params := map[string]any{}
	m.SetIdentifierArgumentFn(params, "7/alice")
	if params["team_id"] != int64(7) || params["user"] != "alice" {
		t.Fatalf("team member identifiers: %v", params)
	}
	if got, err := m.GetExternalNameFn(map[string]any{"team_id": float64(7), "user": "alice"}); err != nil || got != "7/alice" {
		t.Fatalf("team member external name %q, %v", got, err)
	}
}

func TestOrganizationReferences(t *testing.T) {
	pc := GetProviderNamespaced()
	for name, field := range map[string]string{"forgejo_team": "organization", "forgejo_team_member": "team_id", "forgejo_repository": "owner"} {
		if _, ok := pc.Resources[name].References[field]; !ok {
			t.Errorf("%s: %s must be a reference", name, field)
		}
	}
}

func TestTokenExternalName(t *testing.T) {
	e := ExternalNameConfigs["forgejo_personal_access_token"]
	params := map[string]any{}
	e.SetIdentifierArgumentFn(params, "ci-bobteam/pac")
	if params["user"] != "ci-bobteam" || params["name"] != "pac" {
		t.Fatalf("identifiers not set from the external name: %v", params)
	}
	got, err := e.GetExternalNameFn(map[string]any{"user": "ci-bobteam", "name": "pac", "id": float64(9)})
	if err != nil || got != "ci-bobteam/pac" {
		t.Errorf("got %q, %v; want ci-bobteam/pac", got, err)
	}
	missing := []*tfprotov6.Diagnostic{{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to find personal access token by name",
		Detail:  "Personal access token with user 'ci-bobteam' and name 'pac' not found"}}
	if !e.IsNotFoundDiagnosticFn(missing) {
		t.Error("a missing token must read as not found")
	}
	other := []*tfprotov6.Diagnostic{{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to list personal access tokens", Detail: "Unknown error (status 401): x"}}
	if e.IsNotFoundDiagnosticFn(other) {
		t.Error("a failed listing must stay an error")
	}
}

func TestUserNotFound(t *testing.T) {
	e := ExternalNameConfigs["forgejo_user"]
	missing := []*tfprotov6.Diagnostic{{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to read user", Detail: "Unknown error (status 200): user not found with id 0"}}
	if !e.IsNotFoundDiagnosticFn(missing) {
		t.Error("an empty search result must read as not found")
	}
	denied := []*tfprotov6.Diagnostic{{Severity: tfprotov6.DiagnosticSeverityError,
		Summary: "Unable to read user", Detail: "Unknown error (status 403): token does not have at least one of required scope(s): [read:user]"}}
	if e.IsNotFoundDiagnosticFn(denied) {
		t.Error("a denied read must stay an error")
	}
}

func TestOrganizationOwnerExternalName(t *testing.T) {
	e := ExternalNameConfigs["forgejo_organization_owner"]
	params := map[string]any{}
	e.SetIdentifierArgumentFn(params, "acme/alice")
	if params["organization"] != "acme" || params["user"] != "alice" {
		t.Fatalf("identifiers not set: %v", params)
	}
	if got, err := e.GetExternalNameFn(map[string]any{"organization": "acme", "user": "alice"}); err != nil || got != "acme/alice" {
		t.Errorf("got %q, %v", got, err)
	}
}
