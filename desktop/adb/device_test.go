package adb_test

import (
	"testing"
	"time"

	"github.com/sober/desktop/adb"
)

func TestParseDevicesConnected(t *testing.T) {
	output := "List of devices attached\nemulator-5554\tdevice\n"
	devices := adb.ParseDevices(output)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0] != "emulator-5554" {
		t.Errorf("expected emulator-5554, got %s", devices[0])
	}
}

func TestParseDevicesEmpty(t *testing.T) {
	output := "List of devices attached\n"
	devices := adb.ParseDevices(output)
	if len(devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(devices))
	}
}

func TestParseDevicesUnauthorized(t *testing.T) {
	output := "List of devices attached\nemulator-5554\tunauthorized\n"
	devices := adb.ParseDevices(output)
	if len(devices) != 0 {
		t.Fatalf("unauthorized device should not be returned, got %d", len(devices))
	}
}

func TestParseDevicesOffline(t *testing.T) {
	output := "List of devices attached\nemulator-5554\toffline\n"
	devices := adb.ParseDevices(output)
	if len(devices) != 0 {
		t.Fatalf("offline device should not be returned, got %d", len(devices))
	}
}

func TestPollerCallsOnChangeOnConnect(t *testing.T) {
	callCount := 0
	var gotConnected bool
	var gotSerial string

	fake := &fakeRunner{output: "List of devices attached\ntest-serial\tdevice\n"}
	p := adb.NewPoller(fake, 50*time.Millisecond, func(connected bool, serial string) {
		callCount++
		gotConnected = connected
		gotSerial = serial
	})
	p.Start()
	time.Sleep(200 * time.Millisecond)
	p.Stop()

	if callCount == 0 {
		t.Fatal("expected onChange to be called at least once")
	}
	if !gotConnected {
		t.Error("expected connected=true")
	}
	if gotSerial != "test-serial" {
		t.Errorf("expected test-serial, got %s", gotSerial)
	}
}

func TestPollerCallsOnChangeOnDisconnect(t *testing.T) {
	connected := true
	fake := &fakeRunner{output: "List of devices attached\ntest-serial\tdevice\n"}
	p := adb.NewPoller(fake, 50*time.Millisecond, func(c bool, _ string) {
		connected = c
	})
	p.Start()
	time.Sleep(100 * time.Millisecond)
	fake.output = "List of devices attached\n"
	time.Sleep(200 * time.Millisecond)
	p.Stop()

	if connected {
		t.Error("expected connected=false after device removed")
	}
}
