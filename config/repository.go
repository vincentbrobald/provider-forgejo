// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

func configureRepositoryKinds(p *ujconfig.Provider) {
	for name, kind := range map[string]string{
		"forgejo_deploy_key":         "DeployKey",
		"forgejo_repository_webhook": "Webhook",
		"forgejo_collaborator":       "Collaborator",
		"forgejo_branch_protection":  "BranchProtection",
		"forgejo_repository":         "Repository",
		// In this group: it references a Repository and a Team, and the
		// organization group must not reference this one (Go import cycle).
		"forgejo_team_repository": "TeamRepository",
	} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.ShortGroup = "repository"
			r.Kind = kind
		})
	}

	// Upjet types every "id" in the CRDs as a string, but a repository's is a
	// number in the Terraform state: convert it on the way to the status
	// (back to Terraform, a numeric string is accepted as a number).
	p.AddResourceConfigurator("forgejo_repository", func(r *ujconfig.Resource) {
		r.TerraformConversions = append(r.TerraformConversions, idAsString{})
		r.References["owner"] = ujconfig.Reference{TerraformName: "forgejo_organization"}
	})

	for _, name := range []string{"forgejo_deploy_key", "forgejo_repository_webhook", "forgejo_collaborator", "forgejo_branch_protection"} {
		p.AddResourceConfigurator(name, func(r *ujconfig.Resource) {
			r.References["repository_id"] = ujconfig.Reference{TerraformName: "forgejo_repository"}
		})
	}

	p.AddResourceConfigurator("forgejo_team_repository", func(r *ujconfig.Resource) {
		r.References["team_id"] = ujconfig.Reference{TerraformName: "forgejo_team"}
		r.References["repository"] = ujconfig.Reference{
			TerraformName: "forgejo_repository",
			Extractor:     `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("name",false)`,
		}
	})

	// A webhook's "config" map holds its secret ("secret" key) but isn't
	// marked sensitive upstream: as sensitive, the whole map comes from a
	// Secret (spec.forProvider.configSecretRef: keys url, content_type,
	// secret...), never from the spec, so it never lands in Git in clear.
	//
	// Optional for Upjet (the Terraform provider still refuses a webhook
	// without it): as a required sensitive field, Upjet makes
	// initProvider.configSecretRef required too, and every write of the
	// object without it is refused.
	p.AddResourceConfigurator("forgejo_repository_webhook", func(r *ujconfig.Resource) {
		r.TerraformResource.Schema["config"].Sensitive = true
		r.TerraformResource.Schema["config"].Required = false
		r.TerraformResource.Schema["config"].Optional = true
	})
}

type idAsString struct{}

func (idAsString) Convert(params map[string]any, _ *ujconfig.Resource, mode ujconfig.Mode) (map[string]any, error) {
	if mode != ujconfig.FromTerraform {
		return params, nil
	}
	if v, ok := params["id"]; ok && v != nil {
		if _, isString := v.(string); !isString {
			id, err := formatValue(v)
			if err != nil {
				return nil, err
			}
			params["id"] = id
		}
	}
	return params, nil
}
