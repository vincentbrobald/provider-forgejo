// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

// configureUserKinds puts users and their access tokens in the group "user"
// (e.g. users.user.forgejo.m.crossplane.brobald.be).
//
// A User needs a provider whose user is a Forgejo site admin (token scope
// write:admin). A PersonalAccessToken is created as its own user: Forgejo
// makes tokens only for the user who authenticates with a password (basic
// auth), so its ProviderConfig holds that user's name and password. Its
// token is published as attribute.token of the connection Secret.
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
