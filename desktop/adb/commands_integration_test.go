//go:build integration

package adb_test

import (
	"testing"

	"github.com/sober/desktop/adb"
)

func TestListAppsIntegration(t *testing.T) {
	runner, err := adb.NewSystemRunner()
	if err != nil {
		t.Fatalf("NewSystemRunner: %v", err)
	}

	// Diagnostic: show raw broadcast output
	out, err := runner.Run("shell", "am", "broadcast",
		"-a", "com.sober.LIST_APPS",
		"-n", "com.sober.admin/.CommandReceiver",
	)
	t.Logf("raw broadcast output: %q err: %v", out, err)

	// Also check device owner status
	owners, _ := runner.Run("shell", "dpm", "list-owners")
	t.Logf("dpm list-owners: %q", owners)

	c := adb.NewCommands(runner)
	apps, err := c.ListApps()
	if err != nil {
		t.Fatalf("ListApps failed: %v", err)
	}
	if len(apps) == 0 {
		t.Fatal("expected at least one app, got empty list")
	}
	for _, app := range apps {
		t.Logf("  app: %s (%s) hidden=%v", app.Package, app.Label, app.Hidden)
	}
}
