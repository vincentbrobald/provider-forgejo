// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package tfprovider

import (
	"context"
	"fmt"
	"net/http"

	"codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ReadOrganizationOwnerSummary is the summary of the error Read reports; with
// " not found: " in its detail the user isn't an owner (see
// config/external_name.go).
const ReadOrganizationOwnerSummary = "Unable to read organization owner"

// ownersTeam is the name of the team Forgejo makes with every organization,
// whose members own it (it can't be renamed or deleted).
const ownersTeam = "Owners"

var (
	_ resource.Resource              = &organizationOwnerResource{}
	_ resource.ResourceWithConfigure = &organizationOwnerResource{}
)

// NewOrganizationOwnerResource returns forgejo_organization_owner: a member
// of an organization's team Owners. Forgejo makes that team with the
// organization, and the Organization kind doesn't expose its ID, so the team
// is found by its name.
func NewOrganizationOwnerResource() resource.Resource {
	return &organizationOwnerResource{}
}

// organizationOwnerResource implements forgejo_organization_owner.
type organizationOwnerResource struct {
	client organizationOwnerClient
}

// organizationOwnerClient is the part of the Forgejo client the resource uses
// (a fake in the tests).
type organizationOwnerClient interface {
	ListOrgTeams(org string, opt forgejo.ListTeamsOptions) ([]*forgejo.Team, *forgejo.Response, error)
	GetTeamMember(id int64, user string) (*forgejo.User, *forgejo.Response, error)
	AddTeamMember(id int64, user string) (*forgejo.Response, error)
	RemoveTeamMember(id int64, user string) (*forgejo.Response, error)
}

// organizationOwnerModel is the resource's configuration and state.
type organizationOwnerModel struct {
	Organization types.String `tfsdk:"organization"`
	User         types.String `tfsdk:"user"`
}

// Metadata names the resource <provider>_organization_owner.
func (r *organizationOwnerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_owner"
}

// Schema describes the resource: two required strings, each replacing the
// resource when changed.
func (r *organizationOwnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An owner of an organization: a member of its team Owners, found by name.",
		Attributes: map[string]schema.Attribute{
			"organization": schema.StringAttribute{
				Description:   "Name of the organization. Changing this forces a new resource to be created.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"user": schema.StringAttribute{
				Description:   "Name of the user (who must exist). Changing this forces a new resource to be created.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// Configure takes the provider's Forgejo client. Without provider data (as
// during validation, before the provider is configured) it does nothing;
// data of another type is an error.
func (r *organizationOwnerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*forgejo.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *forgejo.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = client
}

// ownersTeamID returns the ID of the organization's team Owners, looked up
// by listing its teams page by page. summary is that of the error to report
// (a missing organization is an error, never "not found": that would make
// Crossplane try to create the membership).
func (r *organizationOwnerResource) ownersTeamID(org, summary string) (int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	for page := 1; ; page++ {
		teams, _, err := r.client.ListOrgTeams(org, forgejo.ListTeamsOptions{
			ListOptions: forgejo.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			diags.AddError(summary, fmt.Sprintf("Teams of organization '%s': %s", org, err))
			return 0, diags
		}
		for _, t := range teams {
			if t != nil && t.Name == ownersTeam {
				return t.ID, diags
			}
		}
		// A short page is the last one.
		if len(teams) < 50 {
			break
		}
	}
	diags.AddError(summary, fmt.Sprintf("Organization '%s' has no team %s", org, ownersTeam))
	return 0, diags
}

// Create adds the user to the organization's team Owners.
func (r *organizationOwnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationOwnerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, user := data.Organization.ValueString(), data.User.ValueString()
	id, diags := r.ownersTeamID(org, "Unable to add organization owner")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.AddTeamMember(id, user); err != nil {
		resp.Diagnostics.AddError("Unable to add organization owner", fmt.Sprintf(
			"User '%s' to team %s (ID %d) of organization '%s': %s", user, ownersTeam, id, org, err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read checks that the user is still an owner (check); the state stays as it
// is.
func (r *organizationOwnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationOwnerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.check(data.Organization.ValueString(), data.User.ValueString())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// check reports an error unless the user is a member of the organization's
// team Owners: with " not found: " in its detail when Forgejo answers 404
// for the membership (see ReadOrganizationOwnerSummary).
func (r *organizationOwnerResource) check(org, user string) diag.Diagnostics {
	id, diags := r.ownersTeamID(org, ReadOrganizationOwnerSummary)
	if diags.HasError() {
		return diags
	}
	if _, res, err := r.client.GetTeamMember(id, user); err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			diags.AddError(ReadOrganizationOwnerSummary, fmt.Sprintf(
				"Owner '%s' of organization '%s' not found: %s", user, org, err))
		} else {
			diags.AddError(ReadOrganizationOwnerSummary, fmt.Sprintf(
				"Owner '%s' of organization '%s': %s", user, org, err))
		}
	}
	return diags
}

// Update only stores the plan: every attribute replaces the resource when
// changed, so there is nothing to change in Forgejo.
func (r *organizationOwnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data organizationOwnerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes the user from the team Owners; a 404 (no longer a member)
// counts as done. Forgejo refuses to remove an organization's last owner.
func (r *organizationOwnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationOwnerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, user := data.Organization.ValueString(), data.User.ValueString()
	id, diags := r.ownersTeamID(org, "Unable to remove organization owner")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	res, err := r.client.RemoveTeamMember(id, user)
	if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
		resp.Diagnostics.AddError("Unable to remove organization owner", fmt.Sprintf(
			"User '%s' from team %s of organization '%s': %s", user, ownersTeam, org, err))
	}
}
