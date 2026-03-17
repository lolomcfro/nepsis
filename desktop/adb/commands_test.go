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
		fake := &callTrackingRunner{
			responses: map[string]string{
				"list-owners":      "{}",
				"set-device-owner": "Success",
			},
		}
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
		fake := &callTrackingRunner{
			responses: map[string]string{
				"list-owners":      "{}",
				"set-device-owner": "Error: already set",
			},
		}
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

	t.Run("sober-admin already device owner", func(t *testing.T) {
		fake := &callTrackingRunner{
			responses: map[string]string{
				"list-owners": "com.sober.admin/.AdminReceiver (User 0)",
			},
		}
		c := adb.NewCommands(fake)
		err := c.SetDeviceOwner()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Accountability Mode is already active") {
			t.Errorf("expected friendly already-active message, got: %v", err)
		}
	})

	t.Run("different app is device owner", func(t *testing.T) {
		fake := &callTrackingRunner{
			responses: map[string]string{
				"list-owners": "com.other.mdm/.AdminReceiver (User 0)",
			},
		}
		c := adb.NewCommands(fake)
		err := c.SetDeviceOwner()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Another app is controlling") {
			t.Errorf("expected other-owner message, got: %v", err)
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

func TestCountGoogleAccounts(t *testing.T) {
	t.Run("returns count when accounts present", func(t *testing.T) {
		fake := &fakeRunner{output: "Account {name=a@gmail.com, type=com.google}\nAccount {name=b@gmail.com, type=com.google}\n"}
		c := adb.NewCommands(fake)
		n, err := c.CountGoogleAccounts()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 2 {
			t.Errorf("expected 2, got %d", n)
		}
	})

	t.Run("returns 0 when no accounts", func(t *testing.T) {
		fake := &fakeRunner{output: "Accounts: 0\n"}
		c := adb.NewCommands(fake)
		n, err := c.CountGoogleAccounts()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 0 {
			t.Errorf("expected 0, got %d", n)
		}
	})

	t.Run("returns 0 on runner error (fail open)", func(t *testing.T) {
		fake := &fakeRunner{err: fmt.Errorf("adb fail")}
		c := adb.NewCommands(fake)
		n, err := c.CountGoogleAccounts()
		if err != nil {
			t.Fatalf("fail-open: expected nil, got %v", err)
		}
		if n != 0 {
			t.Errorf("expected 0 on error, got %d", n)
		}
	})
}

func TestOpenAccountSettings(t *testing.T) {
	fake := &fakeRunner{}
	c := adb.NewCommands(fake)
	err := c.OpenAccountSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	call := strings.Join(fake.calls[0], " ")
	if !strings.Contains(call, "android.settings.SYNC_SETTINGS") {
		t.Errorf("expected SYNC_SETTINGS intent, got: %s", call)
	}
}

func TestExportContacts(t *testing.T) {
	vcf := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Test\r\nEND:VCARD\r\n"

	t.Run("success with contacts", func(t *testing.T) {
		fake := &callTrackingRunner{
			responses: map[string]string{
				"broadcast": "",
				"cat":       vcf,
				"rm":        "",
			},
		}
		c := adb.NewCommands(fake)
		result, err := c.ExportContacts()
		if err != nil {
			t.Fatalf("ExportContacts error: %v", err)
		}
		if !strings.Contains(result, "BEGIN:VCARD") {
			t.Errorf("expected VCF content, got: %s", result)
		}
	})

	t.Run("success with no contacts (empty file)", func(t *testing.T) {
		fake := &callTrackingRunner{
			responses: map[string]string{
				"broadcast": "",
				"cat":       "",
				"rm":        "",
			},
		}
		c := adb.NewCommands(fake)
		result, err := c.ExportContacts()
		if err != nil {
			t.Fatalf("ExportContacts error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string for no contacts, got: %s", result)
		}
	})

	t.Run("broadcast error", func(t *testing.T) {
		fake := &fakeRunner{err: fmt.Errorf("device not found")}
		c := adb.NewCommands(fake)
		_, err := c.ExportContacts()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("device error response", func(t *testing.T) {
		fake := &callTrackingRunner{
			responses: map[string]string{
				"broadcast": "",
				"cat":       `{"error":"permission denied"}`,
				"rm":        "",
			},
		}
		c := adb.NewCommands(fake)
		_, err := c.ExportContacts()
		if err == nil {
			t.Fatal("expected error from device, got nil")
		}
		if !strings.Contains(err.Error(), "EXPORT_CONTACTS failed") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestImportContacts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fake := &callTrackingRunner{
			responses: map[string]string{
				"push":      "",
				"broadcast": "",
				"cat":       `{"success":true,"count":3}`,
				"rm":        "",
			},
		}
		c := adb.NewCommands(fake)
		err := c.ImportContacts("/tmp/backup.vcf")
		if err != nil {
			t.Fatalf("ImportContacts error: %v", err)
		}
	})

	t.Run("push error", func(t *testing.T) {
		fake := &fakeRunner{err: fmt.Errorf("push failed")}
		c := adb.NewCommands(fake)
		err := c.ImportContacts("/tmp/backup.vcf")
		if err == nil {
			t.Fatal("expected error on push failure")
		}
		if !strings.Contains(err.Error(), "push contacts") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("device error response", func(t *testing.T) {
		fake := &callTrackingRunner{
			responses: map[string]string{
				"push":      "",
				"broadcast": "",
				"cat":       `{"error":"file not found"}`,
				"rm":        "",
			},
		}
		c := adb.NewCommands(fake)
		err := c.ImportContacts("/tmp/backup.vcf")
		if err == nil {
			t.Fatal("expected error from device")
		}
		if !strings.Contains(err.Error(), "IMPORT_CONTACTS failed") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestClearDeviceOwner(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// After the broadcast, dpm list-owners returns no sober.admin
		fake := &callTrackingRunner{
			responses: map[string]string{
				"broadcast":   "",
				"list-owners": "{}",  // no device owner
			},
		}
		c := adb.NewCommands(fake)
		err := c.ClearDeviceOwner()
		if err != nil {
			t.Fatalf("ClearDeviceOwner error: %v", err)
		}
	})

	t.Run("broadcast error", func(t *testing.T) {
		fake := &fakeRunner{err: fmt.Errorf("device offline")}
		c := adb.NewCommands(fake)
		err := c.ClearDeviceOwner()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
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
