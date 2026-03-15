package main

import "embed"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/adb/linux/adb
var adbLinux []byte

//go:embed assets/adb/darwin/adb
var adbDarwin []byte

//go:embed assets/adb/windows/adb.exe
var adbWindows []byte

//go:embed assets/sober-admin.apk
var soberAdminAPK []byte

// BundledAdminVersion is the versionCode of the embedded sober-admin.apk.
// Bump this whenever assets/sober-admin.apk is replaced.
const BundledAdminVersion = 1
