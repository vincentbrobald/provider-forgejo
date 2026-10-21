// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestGetProviders(t *testing.T) {
	for _, pc := range []struct {
		root string
		get  func() int
	}{
		{"forgejo.crossplane.brobald.be", func() int { return len(GetProvider().Resources) }},
		{"forgejo.m.crossplane.brobald.be", func() int { return len(GetProviderNamespaced().Resources) }},
	} {
		if n := pc.get(); n != len(ExternalNameConfigs) {
			t.Fatalf("%s: %d kinds, want %d", pc.root, n, len(ExternalNameConfigs))
		}
	}
	p := GetProvider()
	for name, want := range map[string]string{"forgejo_team_repository": "repository", "forgejo_organization_owner": "organization", "forgejo_personal_access_token": "user"} {
		if r := p.Resources[name]; r == nil || r.ShortGroup != want {
			t.Fatalf("%s: not in group %s", name, want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want string
		err  bool
	}{
		{"12", "12", false}, {int64(12), "12", false}, {12, "12", false}, {float64(12), "12", false},
		{big.NewFloat(12), "12", false}, {true, "true", false}, {nil, "", true}, {"", "", true},
	} {
		got, err := formatValue(tc.in)
		if got != tc.want || (err != nil) != tc.err {
			t.Errorf("%#v: %q, %v", tc.in, got, err)
		}
	}
}

func TestNamePairMalformed(t *testing.T) {
	e := namePair("organization", "user", nil)
	for _, n := range []string{"", "acme", "acme/", "/alice"} {
		base := map[string]any{}
		e.SetIdentifierArgumentFn(base, n)
		if len(base) != 0 {
			t.Errorf("%q set %v", n, base)
		}
	}
	if _, err := e.GetExternalNameFn(map[string]any{"organization": "acme"}); err == nil {
		t.Error("no user in the state: an error")
	}
	if _, err := e.GetExternalNameFn(map[string]any{}); err == nil {
		t.Error("nothing in the state: an error")
	}
}

func TestMissingParentIsNotFound(t *testing.T) {
	diag := func(summary, detail string) []*tfprotov6.Diagnostic {
		return []*tfprotov6.Diagnostic{{Severity: tfprotov6.DiagnosticSeverityError, Summary: summary, Detail: detail}}
	}
	for _, tc := range []struct {
		resource, summary, detail string
		want                      bool
	}{
		{"forgejo_repository_webhook", "Unable to read repository", "Repository with ID 29 not found: 404 Not Found", true},
		{"forgejo_repository_webhook", "Unable to read repository webhook", "Repository webhook with owner 'o', repo 'r' and ID 5 not found: 404", true},
		{"forgejo_deploy_key", "Unable to read repository", "Repository with ID 29 not found: 404 Not Found", true},
		{"forgejo_team_member", "Unable to read team", "Team with ID 12 not found: 404 Not Found", true},
		{"forgejo_repository_webhook", "Unable to read repository", "Unknown error (status 500): boom", false},
		{"forgejo_repository_webhook", "Unable to read team", "Team with ID 12 not found: 404", false},
		{"forgejo_team_member", "Unable to read repository", "Repository with ID 29 not found: 404", false},
	} {
		if got := ExternalNameConfigs[tc.resource].IsNotFoundDiagnosticFn(diag(tc.summary, tc.detail)); got != tc.want {
			t.Errorf("%s, %q: %t", tc.resource, tc.summary, got)
		}
	}
}
