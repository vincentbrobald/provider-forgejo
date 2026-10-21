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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ReadTeamRepositorySummary is the summary of the error Read reports; with
// " not found: " in its detail the team doesn't have the repository (see
// config/external_name.go).
const ReadTeamRepositorySummary = "Unable to read team repository"

var (
	_ resource.Resource              = &teamRepositoryResource{}
	_ resource.ResourceWithConfigure = &teamRepositoryResource{}
)

func NewTeamRepositoryResource() resource.Resource {
	return &teamRepositoryResource{}
}

type teamRepositoryResource struct {
	client teamRepositoryClient
}

type teamRepositoryClient interface {
	ListTeamRepositories(id int64, opt forgejo.ListTeamRepositoriesOptions) ([]*forgejo.Repository, *forgejo.Response, error)
	AddTeamRepository(id int64, org, repo string) (*forgejo.Response, error)
	RemoveTeamRepository(id int64, org, repo string) (*forgejo.Response, error)
}

type teamRepositoryModel struct {
	TeamID       types.Int64  `tfsdk:"team_id"`
	Organization types.String `tfsdk:"organization"`
	Repository   types.String `tfsdk:"repository"`
}

func (r *teamRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_repository"
}

func (r *teamRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A repository of an organization given to one of its teams: the team's permission and units apply to it.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.Int64Attribute{
				Description:   "Numeric identifier of the team. Changing this forces a new resource to be created.",
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"organization": schema.StringAttribute{
				Description:   "Name of the organization owning the repository (the team's). Changing this forces a new resource to be created.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository": schema.StringAttribute{
				Description:   "Name of the repository. Changing this forces a new resource to be created.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *teamRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *teamRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data teamRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.AddTeamRepository(data.TeamID.ValueInt64(), data.Organization.ValueString(), data.Repository.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to add team repository", fmt.Sprintf(
			"Repository '%s/%s' to team with ID %d: %s", data.Organization.ValueString(), data.Repository.ValueString(),
			data.TeamID.ValueInt64(), err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *teamRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data teamRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.check(data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Forgejo's API has a direct check (GET /teams/{id}/repos/{org}/{repo}), but
// the SDK has no call for it, so the team's repositories are listed (a team
// has few).
func (r *teamRepositoryResource) check(data teamRepositoryModel) diag.Diagnostics {
	var diags diag.Diagnostics
	id, org, repo := data.TeamID.ValueInt64(), data.Organization.ValueString(), data.Repository.ValueString()
	for page := 1; ; page++ {
		repos, res, err := r.client.ListTeamRepositories(id, forgejo.ListTeamRepositoriesOptions{
			ListOptions: forgejo.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			if res != nil && res.StatusCode == http.StatusNotFound {
				diags.AddError(ReadTeamRepositorySummary, fmt.Sprintf("Team with ID %d not found: %s", id, err))
			} else {
				diags.AddError(ReadTeamRepositorySummary, fmt.Sprintf("Team with ID %d: %s", id, err))
			}
			return diags
		}
		for _, x := range repos {
			if x != nil && x.Name == repo && x.Owner != nil && x.Owner.UserName == org {
				return diags
			}
		}
		if len(repos) < 50 {
			break
		}
	}
	diags.AddError(ReadTeamRepositorySummary, fmt.Sprintf("Repository '%s/%s' in team with ID %d not found: not among its repositories", org, repo, id))
	return diags
}

func (r *teamRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data teamRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *teamRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data teamRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	res, err := r.client.RemoveTeamRepository(data.TeamID.ValueInt64(), data.Organization.ValueString(), data.Repository.ValueString())
	if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
		resp.Diagnostics.AddError("Unable to remove team repository", fmt.Sprintf(
			"Repository '%s/%s' from team with ID %d: %s", data.Organization.ValueString(), data.Repository.ValueString(),
			data.TeamID.ValueInt64(), err))
	}
}
