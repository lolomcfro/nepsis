package adb

// AppManager abstracts hide/show/list/uninstall operations so the UI layer
// is independent of whether Device Owner or direct-ADB mode is active.
// *Commands implements this interface — no adapter needed.
type AppManager interface {
	ListApps() ([]App, error)
	HideApp(pkg string) error
	ShowApp(pkg string) error
	UninstallApp(pkg string) error
}

// var _ AppManager = (*Commands)(nil) is kept permanently — idiomatic Go for
// documenting interface compliance and catching regressions at compile time.
var _ AppManager = (*Commands)(nil)
