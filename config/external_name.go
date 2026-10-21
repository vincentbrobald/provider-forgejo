// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/pkg/errors"

	"brobald.be/provider-forgejo/internal/tfprovider"
)

// ExternalNameConfigs lists the Terraform resources that become kinds, with
// their external name (crossplane.io/external-name).
//
// How each is identified:
//   - repositories, teams and users: their own numeric ID
//     (numericIDExternalName, userExternalName);
//   - organizations: their name (organizationExternalName);
//   - the per-repository resources, team members and team repositories,
//     which have no "id" attribute: "<parent>/<key>", the repository's (or
//     team's) numeric ID and a key (a numeric ID, a user, a branch or
//     repository name; scoped);
//   - access tokens and organization owners: "<a>/<b>" of two names (a
//     token's user and name, an owner's organization and user; namePair).
//
// Upjet reads the external name into the identifying attributes before each
// call, and builds it from the state after a creation. To adopt an existing
// object, set the annotation by hand.
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

// numericIDExternalName is the external name of a repository or a team: its
// numeric ID, which Forgejo assigns ("12"); the resource reads it by that ID.
// Before the creation there is none: the ID "0" then reads as "not found"
// (Forgejo has no object 0), so Crossplane creates it. "id" is computed: it
// never goes into the configuration sent to the Terraform provider.
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

// organizationExternalName is the external name of an organization: its name,
// which the resource reads it by (and which is also spec.forProvider.name).
// Its numeric "id" is computed and unused for reading: the stand-in "0" fills
// it until the first read, as a name isn't a number.
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

// userExternalName is the external name of a user: its numeric ID, like a
// repository's. The client looks the ID up by searching, so a missing user is
// an empty result ("user not found with id 0", status 200), not a 404.
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

// tokenExternalName is the external name of an access token, "<user>/<name>",
// which the resource reads it by (it lists the user's tokens and picks the
// name). Its numeric "id" is computed and unused for reading (stand-in "0", as
// for organizations). A missing token is reported as "Unable to find personal
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

// namePair is the external name "<parent>/<key>" of two names (strings,
// both also in spec.forProvider): a token's user and name, an organization
// owner's organization and user. The resource has no "id" attribute unless
// the caller adds one.
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

// scoped is the external name "<parent>/<key>": the numeric ID of what the
// object belongs to (parent: repository_id, team_id) and its key. numeric: the key
// is a number (an ID Forgejo assigns), else a string the user sets (and
// which is then also in spec.forProvider). readSummary: the summary of the
// error the resource's Read reports when the object is missing; with
// "not found" in its detail it means "does not exist" (so Crossplane creates
// it), not a failure.
func scoped(parent, key string, numeric bool, readSummary string) ujconfig.ExternalName {
	return ujconfig.ExternalName{
		SetIdentifierArgumentFn: func(base map[string]any, externalName string) {
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
		// No "id" attribute: Upjet never asks for one.
		GetIDFn:                ujconfig.ExternalNameAsID,
		DisableNameInitializer: true,
		IsNotFoundDiagnosticFn: notFound(readSummary),
	}
}

// notFound matches the error a resource's Read reports when Forgejo answers
// 404: its summary (readSummary) with "not found" in the detail. Such an
// error means "does not exist" (so Crossplane creates the object), not a
// failure.
func notFound(readSummary string) func([]*tfprotov6.Diagnostic) bool {
	return func(diags []*tfprotov6.Diagnostic) bool {
		for _, d := range diags {
			if d.Severity == tfprotov6.DiagnosticSeverityError && d.Summary == readSummary &&
				strings.Contains(d.Detail, " not found: ") {
				return true
			}
		}
		return false
	}
}

// formatValue renders a state value (a number, as whichever Go type the
// conversion produced, or a string) for the external name.
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

// ExternalNameConfigurations applies the external name of each resource.
func ExternalNameConfigurations() ujconfig.ResourceOption {
	return func(r *ujconfig.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}

// ExternalNameConfigured returns the include list: the configured resources,
// as regular expressions anchored at the end only. No resource name ends with
// another's, so each still matches its own resource alone.
func ExternalNameConfigured() []string {
	l := make([]string, 0, len(ExternalNameConfigs))
	for name := range ExternalNameConfigs {
		l = append(l, name+"$")
	}
	return l
}
