// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

// configureUserKinds puts the user and personal access token resources in
// the group "user" with their kinds, and gives their numeric "id" to the
// status as a string (idAsString).
func configureUserKinds(p *ujconfig.Provider) {
	for name, kind := range map[string]string{
		"forgejo_user":                  "User",
		"forgejo_personal_access_token": "PersonalAccessToken",
	} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.ShortGroup = "user"
			r.Kind = kind
			r.TerraformConversions = append(r.TerraformConversions, idAsString{})
		})
	}
}
