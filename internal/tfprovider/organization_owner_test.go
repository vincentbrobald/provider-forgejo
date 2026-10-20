// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package tfprovider

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v3"
)

type fakeOwners struct {
	teams   []*forgejo.Team
	members map[int64][]string
	listErr bool
}

func (f *fakeOwners) ListOrgTeams(_ string, opt forgejo.ListTeamsOptions) ([]*forgejo.Team, *forgejo.Response, error) {
	if f.listErr {
		return nil, &forgejo.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("404 Not Found")
	}
	start := (opt.Page - 1) * opt.PageSize
	if start >= len(f.teams) {
		return nil, nil, nil
	}
	return f.teams[start:min(start+opt.PageSize, len(f.teams))], nil, nil
}
func (f *fakeOwners) GetTeamMember(id int64, user string) (*forgejo.User, *forgejo.Response, error) {
	for _, u := range f.members[id] {
		if u == user {
			return &forgejo.User{UserName: u}, nil, nil
		}
	}
	return nil, &forgejo.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("404 Not Found")
}
func (f *fakeOwners) AddTeamMember(int64, string) (*forgejo.Response, error)    { return nil, nil }
func (f *fakeOwners) RemoveTeamMember(int64, string) (*forgejo.Response, error) { return nil, nil }

func TestOrganizationOwner(t *testing.T) {
	// Owners on the second page of teams.
	teams := []*forgejo.Team{}
	for i := 0; i < 50; i++ {
		teams = append(teams, &forgejo.Team{ID: int64(100 + i), Name: "team"})
	}
	teams = append(teams, &forgejo.Team{ID: 2, Name: "Owners"})
	r := &organizationOwnerResource{client: &fakeOwners{teams: teams, members: map[int64][]string{2: {"alice"}}}}

	if d := r.check("acme", "alice"); d.HasError() {
		t.Fatalf("alice is an owner: %v", d)
	}
	d := r.check("acme", "someone")
	if !d.HasError() || d[0].Summary() != ReadOrganizationOwnerSummary || !strings.Contains(d[0].Detail(), " not found: ") {
		t.Fatalf("a non-owner must read as not found: %v", d)
	}
	// A missing organization is an error, not "not found" (no creation).
	r = &organizationOwnerResource{client: &fakeOwners{listErr: true}}
	d = r.check("nope", "alice")
	if !d.HasError() || strings.Contains(d[0].Detail(), " not found: ") {
		t.Fatalf("a missing organization must stay an error: %v", d)
	}
}
