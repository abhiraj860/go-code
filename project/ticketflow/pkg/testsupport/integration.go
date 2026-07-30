// Package testsupport holds helpers shared by tests across the workspace.
package testsupport

import (
	"fmt"
	"os"
	"testing"
)

// RequireEnvVar is the environment variable that turns skips into failures.
const RequireEnvVar = "TF_REQUIRE_INTEGRATION"

// SkipOrFail reports that an integration dependency (Postgres, Redis, Mongo)
// is unreachable.
//
// Locally it skips, so `go test ./...` works on a machine with no stack up.
// In CI, where TF_REQUIRE_INTEGRATION is set, it FAILS instead.
//
// The distinction exists because a skipped test is indistinguishable from a
// passing one in a CI summary. Without this, a misconfigured workflow -- a
// missing service container, a wrong port, a changed password -- produces a
// green build in which the most important tests in the project never ran at
// all. That is worse than a red build, because it is confidence without
// coverage.
func SkipOrFail(t *testing.T, format string, args ...any) {
	t.Helper()

	msg := fmt.Sprintf(format, args...)
	if os.Getenv(RequireEnvVar) != "" {
		t.Fatalf("integration dependency unavailable and %s is set: %s", RequireEnvVar, msg)
	}
	t.Skipf("%s (set %s to make this a failure)", msg, RequireEnvVar)
}
