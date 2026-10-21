// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

// configureOrganizationKinds puts the organization, team, team member and
// organization owner resources in the group "organization" with their kinds,
// gives the organization's and the team's numeric "id" to the status as a
// string (idAsString), and makes a team's organization and a team member's
// team_id references to an Organization and a Team.
func configureOrganizationKinds(p *ujconfig.Provider) {
	for name, kind := range map[string]string{
		"forgejo_organization":       "Organization",
		"forgejo_team":               "Team",
		"forgejo_team_member":        "TeamMember",
		"forgejo_organization_owner": "OrganizationOwner",
	} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.ShortGroup = "organization"
			r.Kind = kind
		})
	}

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
