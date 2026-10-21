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

// fakeOwners is a fake organization owner client: teams are listed page by
// page, members maps a team ID to its members, and listErr makes listing fail
// with 404.
type fakeOwners struct {
	teams   []*forgejo.Team
	members map[int64][]string
	listErr bool
}

// ListOrgTeams returns the requested page of teams (empty past the end).
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

// GetTeamMember returns the member, or a 404 error if the user isn't one.
func (f *fakeOwners) GetTeamMember(id int64, user string) (*forgejo.User, *forgejo.Response, error) {
	for _, u := range f.members[id] {
		if u == user {
			return &forgejo.User{UserName: u}, nil, nil
		}
	}
	return nil, &forgejo.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("404 Not Found")
}

// AddTeamMember succeeds without doing anything.
func (f *fakeOwners) AddTeamMember(int64, string) (*forgejo.Response, error) { return nil, nil }

// RemoveTeamMember succeeds without doing anything.
func (f *fakeOwners) RemoveTeamMember(int64, string) (*forgejo.Response, error) { return nil, nil }

// TestOrganizationOwner checks that the team Owners is found on the second
// page, that a non-member reads as not found, and that a missing
// organization stays an error.
func TestOrganizationOwner(t *testing.T) {
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
	r = &organizationOwnerResource{client: &fakeOwners{listErr: true}}
	d = r.check("nope", "alice")
	if !d.HasError() || strings.Contains(d[0].Detail(), " not found: ") {
		t.Fatalf("a missing organization must stay an error: %v", d)
	}
}
