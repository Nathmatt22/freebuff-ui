//go:build !linux

// Freebuff Theme Studio - no AppImage handling outside Linux.
//
// Freebuff ships for macOS as a normal application bundle, so the Linux-only
// extraction path has no counterpart here. These stubs keep findInstallDir
// identical across platforms.
package main

func looksLikeAppImage() bool { return false }

func autoSetupAppImage() (string, error) {
	return "", nil
}

// staleCopyWarning has nothing to say outside Linux, where the patched install
// is the running one.
func staleCopyWarning(ui string) string { return "" }
