package adb

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Runner executes ADB commands using the bundled ADB binary.
type Runner struct {
	path string
}

// adbBinary is set at startup by main package via SetBinary.
var adbBinary []byte

// SetBinary injects the platform-appropriate ADB binary bytes.
// Must be called before NewRunner().
func SetBinary(b []byte) {
	adbBinary = b
}

// NewSystemRunner returns a Runner using the system `adb` binary from PATH.
// Returns an error if `adb` is not found in PATH.
func NewSystemRunner() (*Runner, error) {
	for _, name := range []string{"adb", "adb.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return &Runner{path: path}, nil
		}
	}
	return nil, fmt.Errorf("adb not found in PATH: adb or adb.exe")
}

// NewAutoRunner returns a Runner using the system `adb` if available,
// otherwise extracts and uses the bundled binary.
func NewAutoRunner() (*Runner, error) {
	if r, err := NewSystemRunner(); err == nil {
		return r, nil
	}
	return NewRunner()
}

// NewRunner extracts the bundled ADB binary to a temp file and returns a Runner.
func NewRunner() (*Runner, error) {
	if len(adbBinary) == 0 {
		return nil, fmt.Errorf("ADB binary not set — call adb.SetBinary() first")
	}

	dir, err := os.MkdirTemp("", "sober-adb-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	name := "adb"
	if runtime.GOOS == "windows" {
		name = "adb.exe"
	}
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, adbBinary, 0755); err != nil {
		return nil, fmt.Errorf("write adb binary: %w", err)
	}

	return &Runner{path: path}, nil
}

// Path returns the path to the extracted ADB binary.
func (r *Runner) Path() string {
	return r.path
}

// Run executes an ADB command with the given arguments and returns combined output.
func (r *Runner) Run(args ...string) (string, error) {
	cmd := exec.Command(r.path, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("adb %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
