package adb_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sober/desktop/adb"
)

func TestHideApp(t *testing.T) {
	fake := &fakeRunner{}
	c := adb.NewCommands(fake)
	err := c.HideApp("com.reddit.frontpage")
	if err != nil {
		t.Fatalf("HideApp error: %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("expected 1 ADB call, got %d", len(fake.calls))
	}
	call := strings.Join(fake.calls[0], " ")
	if !strings.Contains(call, "com.sober.HIDE_APP") {
		t.Errorf("expected HIDE_APP broadcast, got: %s", call)
	}
	if !strings.Contains(call, "com.reddit.frontpage") {
		t.Errorf("expected package in broadcast, got: %s", call)
	}
}

func TestShowApp(t *testing.T) {
	fake := &fakeRunner{}
	c := adb.NewCommands(fake)
	err := c.ShowApp("com.reddit.frontpage")
	if err != nil {
		t.Fatalf("ShowApp error: %v", err)
	}
	call := strings.Join(fake.calls[0], " ")
	if !strings.Contains(call, "com.sober.SHOW_APP") {
		t.Errorf("expected SHOW_APP broadcast, got: %s", call)
	}
}

func TestInstallAPK(t *testing.T) {
	fake := &fakeRunner{output: "Success"}
	c := adb.NewCommands(fake)
	err := c.InstallAPK("/tmp/test.apk")
	if err != nil {
		t.Fatalf("InstallAPK error: %v", err)
	}
	call := strings.Join(fake.calls[0], " ")
	if !strings.Contains(call, "install") {
		t.Errorf("expected install command, got: %s", call)
	}
}

func TestInstallAPKFailure(t *testing.T) {
	fake := &fakeRunner{output: "Failure [INSTALL_FAILED_ALREADY_EXISTS]"}
	c := adb.NewCommands(fake)
	err := c.InstallAPK("/tmp/test.apk")
	if err == nil {
		t.Fatal("expected error on install failure, got nil")
	}
}

func TestParseAppList(t *testing.T) {
	jsonStr := `[{"package":"com.android.dialer","label":"Phone","icon":"abc","hidden":false}]`
	apps, err := adb.ParseAppList(jsonStr)
	if err != nil {
		t.Fatalf("ParseAppList error: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
	if apps[0].Package != "com.android.dialer" {
		t.Errorf("wrong package: %s", apps[0].Package)
	}
	if apps[0].Label != "Phone" {
		t.Errorf("wrong label: %s", apps[0].Label)
	}
	if apps[0].Hidden != false {
		t.Errorf("wrong hidden state")
	}
}

func TestListAppsSuccess(t *testing.T) {
	appsJSON := `[{"package":"com.foo","label":"Foo","icon":"","hidden":false}]`
	customFake := &callTrackingRunner{
		responses: map[string]string{
			"broadcast": "",
			"cat":       appsJSON,
			"rm":        "",
		},
	}
	c := adb.NewCommands(customFake)
	apps, err := c.ListApps()
	if err != nil {
		t.Fatalf("ListApps error: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
	if apps[0].Package != "com.foo" {
		t.Errorf("wrong package: %s", apps[0].Package)
	}
}

func TestParseAppListInvalidJSON(t *testing.T) {
	_, err := adb.ParseAppList("not-json")
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

// callTrackingRunner returns preset responses based on which ADB subcommand is called.
type callTrackingRunner struct {
	responses map[string]string
}

func (c *callTrackingRunner) Run(args ...string) (string, error) {
	for _, arg := range args {
		if resp, ok := c.responses[arg]; ok {
			return resp, nil
		}
	}
	return "", fmt.Errorf("unexpected args: %v", args)
}
