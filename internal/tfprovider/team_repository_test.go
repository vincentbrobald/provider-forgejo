// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package tfprovider

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v3"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeClient is a fake team repository client: repos are listed page by
// page; a non-zero status makes listing fail with that status.
type fakeClient struct {
	repos  []*forgejo.Repository
	status int
}

// ListTeamRepositories returns the requested page of repositories (empty
// past the end).
func (f *fakeClient) ListTeamRepositories(_ int64, opt forgejo.ListTeamRepositoriesOptions) ([]*forgejo.Repository, *forgejo.Response, error) {
	if f.status != 0 {
		return nil, &forgejo.Response{Response: &http.Response{StatusCode: f.status}}, errors.New("failed")
	}
	start := (opt.Page - 1) * opt.PageSize
	if start >= len(f.repos) {
		return nil, nil, nil
	}
	end := min(start+opt.PageSize, len(f.repos))
	return f.repos[start:end], nil, nil
}

// AddTeamRepository succeeds without doing anything.
func (f *fakeClient) AddTeamRepository(int64, string, string) (*forgejo.Response, error) {
	return nil, nil
}

// RemoveTeamRepository succeeds without doing anything.
func (f *fakeClient) RemoveTeamRepository(int64, string, string) (*forgejo.Response, error) {
	return nil, nil
}

// repo returns a repository named name, owned by org.
func repo(org, name string) *forgejo.Repository {
	return &forgejo.Repository{Name: name, Owner: &forgejo.User{UserName: org}}
}

// model returns the state of team 3's repository acme/<repo>.
func model(repo string) teamRepositoryModel {
	return teamRepositoryModel{TeamID: types.Int64Value(3), Organization: types.StringValue("acme"), Repository: types.StringValue(repo)}
}

// TestCheck checks check across pages, against an absent repository, one of
// another owner, a missing team and a server error.
func TestCheck(t *testing.T) {
	many := make([]*forgejo.Repository, 0, 120)
	for range 110 {
		many = append(many, repo("acme", "other"))
	}
	many = append(many, repo("other-org", "gitops"), repo("acme", "gitops"))
	for _, tc := range []struct {
		name    string
		client  *fakeClient
		repo    string
		missing bool
		failure bool
	}{
		{"found on a later page", &fakeClient{repos: many}, "gitops", false, false},
		{"absent", &fakeClient{repos: many}, "nope", true, false},
		{"same name, other owner", &fakeClient{repos: []*forgejo.Repository{repo("other-org", "gitops")}}, "gitops", true, false},
		{"team gone", &fakeClient{status: http.StatusNotFound}, "gitops", true, false},
		{"server error", &fakeClient{status: http.StatusInternalServerError}, "gitops", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := (&teamRepositoryResource{client: tc.client}).check(model(tc.repo))
			if !diags.HasError() {
				if tc.missing || tc.failure {
					t.Fatal("no error")
				}
				return
			}
			d := diags.Errors()[0]
			notFound := d.Summary() == ReadTeamRepositorySummary && strings.Contains(d.Detail(), " not found: ")
			if notFound != tc.missing || (!notFound) != tc.failure {
				t.Fatalf("got %q / %q", d.Summary(), d.Detail())
			}
		})
	}
}

// TestResourcesIncludeTeamRepository checks that the provider has our
// resource beside the vendored ones.
func TestResourcesIncludeTeamRepository(t *testing.T) {
	p := New("test")()
	var names []string
	for _, f := range p.Resources(context.Background()) {
		var resp resource.MetadataResponse
		f().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "forgejo"}, &resp)
		names = append(names, resp.TypeName)
	}
	joined := strings.Join(names, ",")
	for _, want := range []string{"forgejo_team_repository", "forgejo_team_member", "forgejo_repository"} {
		if !strings.Contains(joined, want) {
			t.Errorf("%s missing from %s", want, joined)
		}
	}
	var _ provider.Provider = p
}
