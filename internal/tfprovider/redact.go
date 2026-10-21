// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package tfprovider

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
)

// echoed matches how Forgejo repeats a rejected credential in its error: it
// tries a basic-auth password as an access token too, and reports it
// ("access token does not exist [sha: <password>]", "task with token
// "<password>": ...").
var echoed = []*regexp.Regexp{
	regexp.MustCompile(`\[sha: [^\]]*\]`),
	regexp.MustCompile(`token "[^"]*"`),
}

// Configure runs the vendored provider's, then removes the credentials from
// its errors: they end up in the managed resources' status, which whoever may
// view the namespace reads (a team's CI user signs in with a password, and
// until that user exists every sign-in fails with its password echoed).
func (p *withExtras) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	p.Provider.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		return
	}
	// The configured password and token, where set and not empty.
	var secrets []string
	for _, a := range []string{"password", "api_token"} {
		var v *string
		if d := req.Config.GetAttribute(ctx, path.Root(a), &v); !d.HasError() && v != nil && *v != "" {
			secrets = append(secrets, *v)
		}
	}
	resp.Diagnostics = redactDiagnostics(resp.Diagnostics, secrets)
}

// redactDiagnostics returns copies of the diagnostics with the secrets and
// echoed credentials replaced in their summaries and details (redact). Every
// diagnostic that isn't an error becomes a warning; attribute paths are lost.
func redactDiagnostics(in diag.Diagnostics, secrets []string) diag.Diagnostics {
	out := make(diag.Diagnostics, 0, len(in))
	for _, d := range in {
		s, t := redact(d.Summary(), secrets), redact(d.Detail(), secrets)
		if d.Severity() == diag.SeverityError {
			out = append(out, diag.NewErrorDiagnostic(s, t))
		} else {
			out = append(out, diag.NewWarningDiagnostic(s, t))
		}
	}
	return out
}

// redact replaces each secret in s with "<redacted>", then the credentials
// Forgejo echoes (echoed), which also catches a credential not in secrets.
func redact(s string, secrets []string) string {
	for _, v := range secrets {
		s = strings.ReplaceAll(s, v, "<redacted>")
	}
	s = echoed[0].ReplaceAllString(s, "[sha: <redacted>]")
	return echoed[1].ReplaceAllString(s, `token "<redacted>"`)
}
