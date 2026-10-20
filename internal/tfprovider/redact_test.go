// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package tfprovider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestRedact(t *testing.T) {
	detail := "Forgejo Client Error: token is malformed\nuser does not exist [uid: 0, name: ci-x]\n" +
		"access token does not exist [sha: s3cretPassw0rd]\ntask with token \"s3cretPassw0rd\": resource does not exist\n" +
		"the password was s3cretPassw0rd, the token t0ken"
	out := redactDiagnostics(diag.Diagnostics{diag.NewErrorDiagnostic("Unable to create Forgejo API client", detail)},
		[]string{"s3cretPassw0rd", "t0ken"})
	got := out[0].Detail()
	if strings.Contains(got, "s3cret") || strings.Contains(got, "t0ken") {
		t.Fatalf("a credential survived: %q", got)
	}
	if !strings.Contains(got, "user does not exist [uid: 0, name: ci-x]") || out[0].Severity() != diag.SeverityError {
		t.Fatalf("the rest must stay: %q", got)
	}
	// Echoed values the configuration doesn't know (e.g. another field).
	if r := redact(`[sha: abc] token "def"`, nil); r != `[sha: <redacted>] token "<redacted>"` {
		t.Fatalf("got %q", r)
	}
}
