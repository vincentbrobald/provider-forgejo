// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

// configureOrganizationKinds puts organizations, teams, team members and
// organization owners in the group "organization" (e.g.
// teams.organization.forgejo.m.crossplane.brobald.be).
//
// An OrganizationOwner names its organization and user (no team ID). A Team
// names its organization (organization, or organizationRef to an
// Organization); a TeamMember its team (teamId, or teamIdRef to a Team).
func configureOrganizationKinds(p *ujconfig.Provider) {
	for name, kind := range map[string]string{
		"forgejo_organization": "Organization",
		"forgejo_team":         "Team",
		"forgejo_team_member":  "TeamMember",
		// Ours (internal/tfprovider): a member of the organization's team
		// Owners, by name.
		"forgejo_organization_owner": "OrganizationOwner",
	} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.ShortGroup = "organization"
			r.Kind = kind
		})
	}

	// Their numeric "id" (see idAsString).
	for _, name := range []string{"forgejo_organization", "forgejo_team"} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.TerraformConversions = append(r.TerraformConversions, idAsString{})
		})
	}

	p.AddResourceConfigurator("forgejo_team", func(r *ujconfig.Resource) {
		r.References["organization"] = ujconfig.Reference{TerraformName: "forgejo_organization"}
	})
	p.AddResourceConfigurator("forgejo_team_member", func(r *ujconfig.Resource) {
		r.References["team_id"] = ujconfig.Reference{TerraformName: "forgejo_team"}
	})
}
