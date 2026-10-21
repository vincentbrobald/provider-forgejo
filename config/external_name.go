// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

// Package config is the Upjet configuration of the provider: which Terraform
// resources become which kinds, in which groups, with which external names
// and references.
package config

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/pkg/errors"

	"brobald.be/provider-forgejo/internal/tfprovider"
)

// ExternalNameConfigs maps each Terraform resource the provider generates to
// its external-name configuration; resources not in it aren't generated
// (ExternalNameConfigured). External names: a numeric ID (repository, team,
// user), a name (organization), "<repository or team ID>/<key>" for an object
// of a repository or team, "<organization>/<user>" for an owner and
// "<user>/<name>" for a token.
var ExternalNameConfigs = map[string]ujconfig.ExternalName{
	"forgejo_deploy_key":            scoped("repository_id", "key_id", true, "Unable to read deploy key"),
	"forgejo_repository_webhook":    scoped("repository_id", "webhook_id", true, "Unable to read repository webhook"),
	"forgejo_collaborator":          scoped("repository_id", "user", false, "Unable to read collaborator"),
	"forgejo_branch_protection":     scoped("repository_id", "branch_name", false, "Unable to read branch protection"),
	"forgejo_repository":            numericIDExternalName("Unable to read repository"),
	"forgejo_team":                  numericIDExternalName("Unable to read team"),
	"forgejo_team_member":           scoped("team_id", "user", false, "Unable to read team member"),
	"forgejo_team_repository":       scoped("team_id", "repository", false, tfprovider.ReadTeamRepositorySummary),
	"forgejo_organization":          organizationExternalName(),
	"forgejo_user":                  userExternalName(),
	"forgejo_personal_access_token": tokenExternalName(),
	"forgejo_organization_owner":    namePair("organization", "user", notFound(tfprovider.ReadOrganizationOwnerSummary)),
}

// numericIDExternalName is the external name of a resource Forgejo
// identifies by a numeric ID: the computed attribute "id". Before the
// creation there is none: the ID "0" then reads as "not found" (Forgejo has
// no object 0), so Crossplane creates it. readSummary is the summary of the
// error the resource's Read reports.
func numericIDExternalName(readSummary string) ujconfig.ExternalName {
	return ujconfig.ExternalName{
		SetIdentifierArgumentFn: ujconfig.NopSetIdentifierArgument,
		GetExternalNameFn: func(tfstate map[string]any) (string, error) {
			id, err := formatValue(tfstate["id"])
			if err != nil {
				return "", errors.Wrap(err, "id")
			}
			// 0 is the stand-in before the creation (GetIDFn), never an
			// object: Upjet would otherwise "recover" it as the external name
			// after a failed creation, and references would resolve to it.
			if id == "0" {
				return "", errors.New("id: not created yet")
			}
			return id, nil
		},
		GetIDFn: func(_ context.Context, externalName string, _ map[string]any, _ map[string]any) (string, error) {
			if externalName == "" {
				return "0", nil
			}
			return externalName, nil
		},
		DisableNameInitializer: true,
		IsNotFoundDiagnosticFn: notFound(readSummary),
		TFPluginFrameworkOptions: ujconfig.TFPluginFrameworkOptions{
			ComputedIdentifierAttributes: []string{"id"},
		},
	}
}

// organizationExternalName is the external name of an organization: its
// name. Its numeric "id" is computed and unused for reading: the stand-in "0"
// fills it until the first read, as a name isn't a number. An empty external
// name leaves the spec's name.
func organizationExternalName() ujconfig.ExternalName {
	return ujconfig.ExternalName{
		SetIdentifierArgumentFn: func(base map[string]any, externalName string) {
			if externalName != "" {
				base["name"] = externalName
			}
		},
		GetExternalNameFn: func(tfstate map[string]any) (string, error) {
			name, err := formatValue(tfstate["name"])
			return name, errors.Wrap(err, "name")
		},
		GetIDFn: func(_ context.Context, _ string, _ map[string]any, _ map[string]any) (string, error) {
			return "0", nil
		},
		DisableNameInitializer: true,
		IsNotFoundDiagnosticFn: notFound("Unable to read organization"),
		TFPluginFrameworkOptions: ujconfig.TFPluginFrameworkOptions{
			ComputedIdentifierAttributes: []string{"id"},
		},
	}
}

// userExternalName is the numeric ID external name of a user, which also
// reads a search without result as "not found": the client looks the ID up
// by searching, so a missing user is an empty result ("user not found with id
// 0", status 200), not a 404.
func userExternalName() ujconfig.ExternalName {
	e := numericIDExternalName("Unable to read user")
	notFound404 := e.IsNotFoundDiagnosticFn
	e.IsNotFoundDiagnosticFn = func(diags []*tfprotov6.Diagnostic) bool {
		for _, d := range diags {
			if d.Severity == tfprotov6.DiagnosticSeverityError && d.Summary == "Unable to read user" &&
				strings.Contains(d.Detail, "user not found with id ") {
				return true
			}
		}
		return notFound404(diags)
	}
	return e
}

// tokenExternalName is the external name of a personal access token:
// "<user>/<name>". Its numeric "id" is computed and unused for reading
// (stand-in "0", as for organizations). A missing token is reported as "Unable to find personal
// access token by name", without the usual "not found: " detail.
func tokenExternalName() ujconfig.ExternalName {
	e := namePair("user", "name", func(diags []*tfprotov6.Diagnostic) bool {
		for _, d := range diags {
			if d.Severity == tfprotov6.DiagnosticSeverityError &&
				d.Summary == "Unable to find personal access token by name" {
				return true
			}
		}
		return false
	})
	e.GetIDFn = func(_ context.Context, _ string, _ map[string]any, _ map[string]any) (string, error) {
		return "0", nil
	}
	e.TFPluginFrameworkOptions = ujconfig.TFPluginFrameworkOptions{ComputedIdentifierAttributes: []string{"id"}}
	return e
}

// namePair is an external name "<parent>/<key>" made of two string
// attributes, parent and key; the key may hold slashes, the parent can't. A
// malformed external name (no slash, an empty part) sets nothing, leaving
// the spec's values. isNotFound tells the diagnostics of a missing object.
func namePair(parent, key string, isNotFound func([]*tfprotov6.Diagnostic) bool) ujconfig.ExternalName {
	return ujconfig.ExternalName{
		SetIdentifierArgumentFn: func(base map[string]any, externalName string) {
			p, k, ok := strings.Cut(externalName, "/")
			if !ok || p == "" || k == "" {
				return
			}
			base[parent] = p
			base[key] = k
		},
		GetExternalNameFn: func(tfstate map[string]any) (string, error) {
			p, err := formatValue(tfstate[parent])
			if err != nil {
				return "", errors.Wrap(err, parent)
			}
			k, err := formatValue(tfstate[key])
			if err != nil {
				return "", errors.Wrap(err, key)
			}
			return p + "/" + k, nil
		},
		GetIDFn:                ujconfig.ExternalNameAsID,
		DisableNameInitializer: true,
		IsNotFoundDiagnosticFn: isNotFound,
	}
}

// scoped is the external name "<parent ID>/<key>" of an object in a
// repository or team: parent is the attribute with the numeric ID
// (repository_id, team_id), key the object's own attribute, a numeric ID if
// numeric is set, otherwise a string that may hold slashes (a branch name).
// A part that isn't a number where one is expected is left unset. readSummary
// is the summary of the error the object's Read reports.
//
// A missing parent means the object doesn't exist either: the resources read
// their repository (or team) first, and report its absence under the
// parent's summary. Otherwise an object whose repository was deleted first
// could never be deleted (its Observe would fail forever); creating one in a
// missing repository still fails, visibly.
func scoped(parent, key string, numeric bool, readSummary string) ujconfig.ExternalName {
	return ujconfig.ExternalName{
		SetIdentifierArgumentFn: func(base map[string]any, externalName string) {
			// Cut at the first slash: the parent ID has none, a string key
			// may.
			repo, rest, ok := strings.Cut(externalName, "/")
			if !ok || rest == "" {
				return
			}
			if id, err := strconv.ParseInt(repo, 10, 64); err == nil {
				base[parent] = id
			}
			if !numeric {
				base[key] = rest
			} else if id, err := strconv.ParseInt(rest, 10, 64); err == nil {
				base[key] = id
			}
		},
		GetExternalNameFn: func(tfstate map[string]any) (string, error) {
			repo, err := formatValue(tfstate[parent])
			if err != nil {
				return "", errors.Wrap(err, parent)
			}
			k, err := formatValue(tfstate[key])
			if err != nil {
				return "", errors.Wrap(err, key)
			}
			return repo + "/" + k, nil
		},
		GetIDFn:                ujconfig.ExternalNameAsID,
		DisableNameInitializer: true,
		IsNotFoundDiagnosticFn: notFound(readSummary, parentReadSummary[parent]),
	}
}

// parentReadSummary is the summary of the error a resource reports when its
// parent is missing (the vendored provider's getRepositoryByID and
// getOrgTeamByID).
var parentReadSummary = map[string]string{
	"repository_id": "Unable to read repository",
	"team_id":       "Unable to read team",
}

// notFound matches the error a resource's Read reports when Forgejo answers
// 404: one of its summaries (readSummaries) with "not found" in the detail.
func notFound(readSummaries ...string) func([]*tfprotov6.Diagnostic) bool {
	return func(diags []*tfprotov6.Diagnostic) bool {
		for _, d := range diags {
			if d.Severity == tfprotov6.DiagnosticSeverityError && slices.Contains(readSummaries, d.Summary) &&
				strings.Contains(d.Detail, " not found: ") {
				return true
			}
		}
		return false
	}
}

// formatValue renders an identifier from the Terraform state as a string:
// a string as it is, a number (int64, int, float64, *big.Float) as a decimal
// integer (a fraction is dropped), anything else with fmt.Sprint. A missing
// (nil) value or an empty string is an error.
func formatValue(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", errors.New("missing in the state")
	case string:
		if x == "" {
			return "", errors.New("empty in the state")
		}
		return x, nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case int:
		return strconv.Itoa(x), nil
	case float64:
		return strconv.FormatInt(int64(x), 10), nil
	case *big.Float:
		i, _ := x.Int64()
		return strconv.FormatInt(i, 10), nil
	default:
		return fmt.Sprint(x), nil
	}
}

// ExternalNameConfigurations returns the resource option that gives each
// resource its configuration from ExternalNameConfigs; the others keep
// Upjet's default.
func ExternalNameConfigurations() ujconfig.ResourceOption {
	return func(r *ujconfig.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}

// ExternalNameConfigured returns the regular expressions of the Terraform
// resources to generate: "<name>$" for each resource of ExternalNameConfigs,
// in no particular order. No resource name ends with another's, so each
// still matches its own resource alone.
func ExternalNameConfigured() []string {
	l := make([]string, 0, len(ExternalNameConfigs))
	for name := range ExternalNameConfigs {
		l = append(l, name+"$")
	}
	return l
}
