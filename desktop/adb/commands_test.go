package adb_test

import (
	"errors"
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

func TestListAppsNoReceiver(t *testing.T) {
	// Simulate broadcast error (e.g. component not found → adb exits non-zero).
	fake := &fakeRunner{err: fmt.Errorf("exit status 1\nError: Broadcast receiver not found")}
	c := adb.NewCommands(fake)
	_, err := c.ListApps()
	if err == nil {
		t.Fatal("expected error when broadcast fails, got nil")
	}
}

func TestListAppsSuccess(t *testing.T) {
	appsJSON := `[{"package":"com.foo","label":"Foo","icon":"","hidden":false}]`
	customFake := &callTrackingRunner{
		responses: map[string]string{
			"list-owners": "com.sober.admin/.AdminReceiver", // satisfies device-owner pre-check
			"broadcast":   "",
			"cat":         appsJSON,
			"rm":          "",
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

func TestApplyRestrictions(t *testing.T) {
	fake := &fakeRunner{}
	c := adb.NewCommands(fake)
	err := c.ApplyRestrictions()
	if err != nil {
		t.Fatalf("ApplyRestrictions error: %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("expected 1 ADB call, got %d", len(fake.calls))
	}
	call := strings.Join(fake.calls[0], " ")
	if !strings.Contains(call, "com.sober.APPLY_RESTRICTIONS") {
		t.Errorf("expected APPLY_RESTRICTIONS broadcast, got: %s", call)
	}
}

func TestCheckAccounts(t *testing.T) {
	t.Run("no accounts", func(t *testing.T) {
		fake := &fakeRunner{output: "Accounts: 0\nServiceInfo: AuthenticatorDescription {type=com.google}, ComponentInfo{com.google.android.gms/...}"}
		c := adb.NewCommands(fake)
		if err := c.CheckAccounts(); err != nil {
			t.Fatalf("expected nil, got: %v", err)
		}
	})

	t.Run("accounts present", func(t *testing.T) {
		fake := &fakeRunner{output: "Account {name=test@gmail.com, type=com.google}\n"}
		c := adb.NewCommands(fake)
		err := c.CheckAccounts()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Google accounts are still on this device") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("runner error", func(t *testing.T) {
		fake := &fakeRunner{err: errors.New("adb fail")}
		c := adb.NewCommands(fake)
		if err := c.CheckAccounts(); err != nil {
			t.Fatalf("expected nil on runner error (fail open), got: %v", err)
		}
	})
}

func TestSetDeviceOwner(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fake := &fakeRunner{output: "Success"}
		c := adb.NewCommands(fake)
		if err := c.SetDeviceOwner(); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("runner error", func(t *testing.T) {
		fake := &fakeRunner{err: errors.New("adb fail")}
		c := adb.NewCommands(fake)
		if err := c.SetDeviceOwner(); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("output contains error", func(t *testing.T) {
		fake := &fakeRunner{output: "Error: already set"}
		c := adb.NewCommands(fake)
		if err := c.SetDeviceOwner(); err == nil {
			t.Fatal("expected error for error output, got nil")
		}
	})

	t.Run("accounts on device", func(t *testing.T) {
		fake := &fakeRunner{err: errors.New("exit status 255\njava.lang.IllegalStateException: there are already some accounts on the device")}
		c := adb.NewCommands(fake)
		err := c.SetDeviceOwner()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Google accounts are still on this device") {
			t.Errorf("expected friendly message, got: %v", err)
		}
	})
}

func TestIsDeviceOwnerInstalled(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		fake := &fakeRunner{output: "com.sober.admin/.AdminReceiver"}
		c := adb.NewCommands(fake)
		if !c.IsDeviceOwnerInstalled() {
			t.Fatal("expected true, got false")
		}
	})

	t.Run("not found", func(t *testing.T) {
		fake := &fakeRunner{output: "{}"}
		c := adb.NewCommands(fake)
		if c.IsDeviceOwnerInstalled() {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("runner error", func(t *testing.T) {
		fake := &fakeRunner{err: errors.New("fail")}
		c := adb.NewCommands(fake)
		if c.IsDeviceOwnerInstalled() {
			t.Fatal("expected false on error, got true")
		}
	})
}

func TestGetInstalledAdminVersionCode_found(t *testing.T) {
	output := "    versionCode=7 targetSdk=33\n    versionName=1.0\n"
	fake := &fakeRunner{output: output}
	c := adb.NewCommands(fake)
	v, err := c.GetInstalledAdminVersionCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 7 {
		t.Errorf("expected versionCode 7, got %d", v)
	}
}

func TestGetInstalledAdminVersionCode_notInstalled(t *testing.T) {
	fake := &fakeRunner{output: ""}
	c := adb.NewCommands(fake)
	v, err := c.GetInstalledAdminVersionCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 0 {
		t.Errorf("expected 0 for not-installed, got %d", v)
	}
}

func TestGetInstalledAdminVersionCode_runnerError(t *testing.T) {
	fake := &fakeRunner{err: errors.New("adb fail")}
	c := adb.NewCommands(fake)
	v, err := c.GetInstalledAdminVersionCode()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if v != 0 {
		t.Errorf("expected 0 on error, got %d", v)
	}
}

func TestGetInstalledAdminVersionCode_verifyArgs(t *testing.T) {
	output := "    versionCode=3 targetSdk=33\n"
	fake := &callTrackingRunner{
		responses: map[string]string{
			"com.sober.admin": output,
		},
	}
	c := adb.NewCommands(fake)
	v, err := c.GetInstalledAdminVersionCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 3 {
		t.Errorf("expected versionCode 3, got %d", v)
	}
}

func TestUninstallApp(t *testing.T) {
	fake := &fakeRunner{}
	c := adb.NewCommands(fake)
	err := c.UninstallApp("com.example.app")
	if err != nil {
		t.Fatalf("UninstallApp error: %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("expected 1 ADB call, got %d", len(fake.calls))
	}
	call := fake.calls[0]
	expectedArgs := []string{"shell", "pm", "uninstall", "--user", "0", "com.example.app"}
	if len(call) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d: %v", len(expectedArgs), len(call), call)
	}
	for i, expected := range expectedArgs {
		if call[i] != expected {
			t.Errorf("arg %d: expected %q, got %q", i, expected, call[i])
		}
	}
}

func TestUninstallAppError(t *testing.T) {
	fake := &fakeRunner{err: errors.New("device not found")}
	c := adb.NewCommands(fake)
	err := c.UninstallApp("com.example.app")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "uninstall com.example.app") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestGetKnownStoreList(t *testing.T) {
	stores := adb.GetKnownStoreList()
	if len(stores) == 0 {
		t.Fatal("expected non-empty store list, got empty")
	}
	found := false
	for _, store := range stores {
		if store == "com.android.vending" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected com.android.vending in store list, got: %v", stores)
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
