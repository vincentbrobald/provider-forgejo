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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// calls records what a resource asked of Forgejo; err fails every write.
type calls struct {
	added, removed []string
	err            error
	status         int // of err's response
}

func (c *calls) res() *forgejo.Response {
	if c.status == 0 {
		return nil
	}
	return &forgejo.Response{Response: &http.Response{StatusCode: c.status}}
}

// teamRepos is a team's repositories that records writes.
type teamRepos struct {
	fakeClient
	calls
}

func (f *teamRepos) AddTeamRepository(id int64, org, repo string) (*forgejo.Response, error) {
	f.added = append(f.added, org+"/"+repo)
	return f.res(), f.err
}
func (f *teamRepos) RemoveTeamRepository(id int64, org, repo string) (*forgejo.Response, error) {
	f.removed = append(f.removed, org+"/"+repo)
	return f.res(), f.err
}

// owners is an organization's teams that records writes.
type owners struct {
	fakeOwners
	calls
}

func (f *owners) AddTeamMember(id int64, user string) (*forgejo.Response, error) {
	f.added = append(f.added, user)
	return f.res(), f.err
}
func (f *owners) RemoveTeamMember(id int64, user string) (*forgejo.Response, error) {
	f.removed = append(f.removed, user)
	return f.res(), f.err
}

// crud runs a resource's Create, Read, Update and Delete with value as the
// plan and the state, and returns each one's diagnostics.
func crud(t *testing.T, r resource.Resource, value any) (create, read, update, del []string) {
	t.Helper()
	ctx := context.Background()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	if sr.Diagnostics.HasError() {
		t.Fatal(sr.Diagnostics)
	}
	s := sr.Schema
	st := tfsdk.State{Schema: s}
	if d := st.Set(ctx, value); d.HasError() {
		t.Fatal(d)
	}
	empty := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}

	cr := resource.CreateResponse{State: empty}
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: s, Raw: st.Raw}}, &cr)
	rr := resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, &rr)
	ur := resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: s, Raw: st.Raw}, State: st}, &ur)
	dr := resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, &dr)
	return details(cr.Diagnostics.Errors()), details(rr.Diagnostics.Errors()),
		details(ur.Diagnostics.Errors()), details(dr.Diagnostics.Errors())
}

// details renders diagnostics as "summary: detail".
func details[T interface {
	Summary() string
	Detail() string
}](ds []T) []string {
	out := []string{}
	for _, d := range ds {
		out = append(out, d.Summary()+": "+d.Detail())
	}
	return out
}

func TestTeamRepositoryCRUD(t *testing.T) {
	f := &teamRepos{fakeClient: fakeClient{repos: []*forgejo.Repository{repo("acme", "gitops")}}}
	c, r, u, d := crud(t, &teamRepositoryResource{client: f}, model("gitops"))
	if len(c)+len(r)+len(u)+len(d) != 0 {
		t.Fatalf("errors: %v %v %v %v", c, r, u, d)
	}
	if strings.Join(f.added, ",") != "acme/gitops" || strings.Join(f.removed, ",") != "acme/gitops" {
		t.Fatalf("added %v, removed %v", f.added, f.removed)
	}

	// Failures are reported; a repository already removed (404) is not one.
	f = &teamRepos{calls: calls{err: errors.New("boom"), status: http.StatusInternalServerError}}
	c, r, _, d = crud(t, &teamRepositoryResource{client: f}, model("gitops"))
	if len(c) != 1 || !strings.Contains(c[0], "Unable to add team repository") || len(d) != 1 || len(r) != 1 {
		t.Fatalf("create %v, read %v, delete %v", c, r, d)
	}
	f = &teamRepos{calls: calls{err: errors.New("404"), status: http.StatusNotFound}}
	if _, _, _, d = crud(t, &teamRepositoryResource{client: f}, model("gitops")); len(d) != 0 {
		t.Fatalf("removing a repository the team no longer has: %v", d)
	}
}

func TestOrganizationOwnerCRUD(t *testing.T) {
	ownersTeam := []*forgejo.Team{{ID: 2, Name: "Owners"}}
	m := organizationOwnerModel{Organization: types.StringValue("acme"), User: types.StringValue("alice")}
	f := &owners{fakeOwners: fakeOwners{teams: ownersTeam, members: map[int64][]string{2: {"alice"}}}}
	c, r, u, d := crud(t, &organizationOwnerResource{client: f}, m)
	if len(c)+len(r)+len(u)+len(d) != 0 {
		t.Fatalf("errors: %v %v %v %v", c, r, u, d)
	}
	if strings.Join(f.added, ",") != "alice" || strings.Join(f.removed, ",") != "alice" {
		t.Fatalf("added %v, removed %v", f.added, f.removed)
	}

	// Forgejo refuses to remove the last owner: the error is reported.
	f = &owners{fakeOwners: fakeOwners{teams: ownersTeam}, calls: calls{err: errors.New("last owner"), status: http.StatusConflict}}
	c, r, _, d = crud(t, &organizationOwnerResource{client: f}, m)
	if len(c) != 1 || len(d) != 1 || !strings.Contains(d[0], "Unable to remove organization owner") {
		t.Fatalf("create %v, delete %v", c, d)
	}
	if len(r) != 1 || !strings.Contains(r[0], " not found: ") {
		t.Fatalf("not a member: not found: %v", r)
	}

	// An organization without its team Owners: every call fails, never "not found".
	f = &owners{fakeOwners: fakeOwners{teams: []*forgejo.Team{{ID: 5, Name: "Devs"}}}}
	c, r, _, d = crud(t, &organizationOwnerResource{client: f}, m)
	for _, e := range [][]string{c, r, d} {
		if len(e) != 1 || !strings.Contains(e[0], "has no team Owners") {
			t.Fatalf("no team Owners: %v", e)
		}
	}
	if len(f.added)+len(f.removed) != 0 {
		t.Fatal("nothing may be written without the team")
	}
}

func TestSchemasAndConfigure(t *testing.T) {
	ctx := context.Background()
	for _, r := range []resource.ResourceWithConfigure{&teamRepositoryResource{}, &organizationOwnerResource{}} {
		var sr resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sr)
		for name, a := range sr.Schema.Attributes {
			if !a.IsRequired() || a.GetDescription() == "" {
				t.Errorf("%s: required, with a description", name)
			}
		}
		var resp resource.ConfigureResponse
		r.Configure(ctx, resource.ConfigureRequest{}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal("no provider data yet (validation): no error")
		}
		r.Configure(ctx, resource.ConfigureRequest{ProviderData: "not a client"}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("another provider data type must be an error")
		}
		resp = resource.ConfigureResponse{}
		r.Configure(ctx, resource.ConfigureRequest{ProviderData: &forgejo.Client{}}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
	}
}

// Configure removes the password from the vendored provider's errors.
func TestConfigureRedacts(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	var sr provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &sr)
	type cfg struct {
		Host     types.String `tfsdk:"host"`
		Username types.String `tfsdk:"username"`
		Password types.String `tfsdk:"password"`
		APIToken types.String `tfsdk:"api_token"`
	}
	st := tfsdk.State{Schema: sr.Schema}
	if d := st.Set(ctx, cfg{Host: types.StringValue("https://127.0.0.1:1"), Username: types.StringValue("ci"),
		Password: types.StringValue("s3cretPassw0rd"), APIToken: types.StringNull()}); d.HasError() {
		t.Fatal(d)
	}
	var resp provider.ConfigureResponse
	p.Configure(ctx, provider.ConfigureRequest{Config: tfsdk.Config{Schema: sr.Schema, Raw: st.Raw}}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("no Forgejo there: the configuration must fail")
	}
	for _, d := range resp.Diagnostics {
		if strings.Contains(d.Summary()+d.Detail(), "s3cretPassw0rd") {
			t.Fatalf("the password survived: %q", d.Detail())
		}
	}
}
