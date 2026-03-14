package adb_test

import (
	"testing"

	"github.com/sober/desktop/adb"
)

func TestRunnerExtractsBinary(t *testing.T) {
	adb.SetBinary([]byte("fake-adb-content"))
	r, err := adb.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner() error: %v", err)
	}
	if r.Path() == "" {
		t.Fatal("runner path should not be empty after extraction")
	}
}
