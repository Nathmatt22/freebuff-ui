//go:build linux

// Freebuff Theme Studio - automatic AppImage setup.
//
// Asking a user to run three commands by hand is a poor experience, and the
// AppImage is the only shape Freebuff ships for Linux. This file does the work
// instead: it finds the AppImage, extracts it once into a writable directory,
// and hands back a path the injector can actually write to.
//
// The starting point is $APPIMAGE, which the AppImage runtime exports to the
// running process. Reading /proc/<pid>/environ therefore locates the exact file
// the user launched, without guessing at download folders.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// appStateDir holds the extracted copy. It belongs under XDG data, not the
// config dir, because it is hundreds of megabytes of application files.
func appStateDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "FreebuffThemeStudio")
}

// extractedRoot is the install the panel is written into.
func extractedRoot() string {
	dir := appStateDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "squashfs-root")
}

// runningAppImage returns the path of the AppImage a running Freebuff came
// from, read from its own environment. It is empty when Freebuff is not
// running as an AppImage, or when the process cannot be read.
func runningAppImage() string {
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return ""
	}
	for _, p := range procs {
		if !p.IsDir() || !isNumericName(p.Name()) {
			continue
		}
		comm := strings.ToLower(processNameFromProc(p.Name()))
		if !strings.Contains(comm, "freebuff") && !strings.Contains(comm, "codebuff") {
			continue
		}
		if app := appImageEnvOf(p.Name()); app != "" {
			return app
		}
	}
	return ""
}

// appImageEnvOf pulls $APPIMAGE out of one process' environment. The kernel
// exposes it as NUL-separated entries; a second, space-separated form is
// accepted because some kernels render it that way.
func appImageEnvOf(pid string) string {
	b, err := os.ReadFile(filepath.Join("/proc", pid, "environ"))
	if err != nil {
		return ""
	}
	for _, entry := range strings.Split(string(b), "\x00") {
		if strings.HasPrefix(entry, "APPIMAGE=") {
			return strings.TrimSpace(strings.TrimPrefix(entry, "APPIMAGE="))
		}
	}
	return ""
}

// searchAppImage looks for an AppImage when Freebuff is not running, so the
// setup can run before the app has ever been started.
func searchAppImage() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dirs := []string{
		"",
		"Téléchargements", "Downloads", "Bureau", "Desktop",
		filepath.Join("Applications"),
	}
	for _, d := range dirs {
		dir := filepath.Join(home, d)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				continue
			}
			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".appimage") {
				continue
			}
			if strings.Contains(lower, "freebuff") || strings.Contains(lower, "codebuff") {
				return filepath.Join(dir, name)
			}
		}
	}
	return ""
}

// appImageFile locates the AppImage to extract, preferring the running one.
func appImageFile() string {
	if p := runningAppImage(); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return searchAppImage()
}

// extractAppImage unpacks the AppImage into the state directory and returns the
// resulting squashfs-root.
//
// The runtime has no flag to choose an output directory, so the extraction is
// run with the state directory as its working directory; it always writes
// ./squashfs-root there.
func extractAppImage(appImage string, progress func(string, ...any)) (string, error) {
	if root := extractedRoot(); isInstallDir(root) {
		return root, nil // already extracted
	}
	dir := appStateDir()
	if dir == "" {
		return "", fmt.Errorf("could not work out where to put the extracted app")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}

	progress("Extracting %s", filepath.Base(appImage))
	progress("This happens once and takes a little while.")

	cmd := exec.Command(appImage, "--appimage-extract")
	cmd.Dir = dir
	hideProc(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// The AppImage runtime is only available if the file kept its execute
		// bit, which is not the case for a fresh download.
		if isExecFormat(string(out)) {
			return "", fmt.Errorf("%s is not executable yet.\n\n"+
				"       Run this once, then run the injector again:\n"+
				"           chmod +x %s\n", appImage, appImage)
		}
		return "", fmt.Errorf("extracting the AppImage: %v\n%s", err, tailLines(string(out), 4))
	}

	root := extractedRoot()
	if !isInstallDir(root) {
		return "", fmt.Errorf("the AppImage extracted, but %s does not look like a Freebuff install", root)
	}
	return root, nil
}

func isExecFormat(out string) bool {
	l := strings.ToLower(out)
	return strings.Contains(l, "exec format error") || strings.Contains(l, "cannot execute binary file") || strings.Contains(l, "permission denied")
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// appImageShortcut writes a desktop entry that starts the extracted copy, so
// the app menu keeps launching the version the panel is installed into. Without
// this the user would launch the AppImage from the menu and get a Freebuff
// without the panel, which is the failure mode worth designing out.
func appImageShortcut(root string) error {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, "applications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	exe := relaunchExe(root)
	if exe == "" {
		return fmt.Errorf("no launcher found inside %s", root)
	}
	body := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Freebuff\n" +
		"Comment=Freebuff Desktop (with Theme Studio installed)\n" +
		"Exec=" + exe + " %U\n" +
		"Terminal=false\n" +
		"Categories=Development;\n"
	return atomicWriteFile(filepath.Join(dir, "freebuff-themed.desktop"), []byte(body), 0o644)
}

// appImageReady reports whether the extracted copy is already set up, so the
// common second run can say so and skip straight to installing.
func appImageReady() bool {
	root := extractedRoot()
	return root != "" && isInstallDir(root)
}

// appImageStale reports that Freebuff was launched from the AppImage after the
// panel was installed into the extracted copy. The two are separate trees, so
// the panel would appear to have vanished.
func appImageStale() bool {
	if !appImageReady() {
		return false
	}
	ui := filepath.Join(extractedRoot(), "resources", "orchestrator", "ui")
	html, err := os.ReadFile(indexPath(ui))
	if err != nil {
		return true
	}
	_, _, injected, err := markerBounds(string(html))
	if err != nil {
		return true
	}
	return !injected
}

// looksLikeAppImage reports whether this machine has a Freebuff AppImage, which
// is the signal that the automatic setup should run.
func looksLikeAppImage() bool {
	if appImageReady() {
		return true
	}
	return appImageFile() != ""
}

// autoSetupAppImage turns an AppImage into a writable install and returns it.
//
// This is what the user runs instead of three commands: the AppImage is found,
// extracted once into the state directory, and the launcher the panel needs is
// created. Extraction is skipped when it has already happened, so running the
// injector a second time costs nothing.
func autoSetupAppImage() (string, error) {
	if appImageReady() {
		root := extractedRoot()
		// Keep the launcher in step even on repeat runs, so a Freebuff update
		// that changes the extracted layout does not leave it dangling.
		if err := appImageShortcut(root); err != nil {
			info("Could not write the menu shortcut: %v", err)
		}
		return root, nil
	}
	appImage := appImageFile()
	if appImage == "" {
		return "", errors.New("no Freebuff AppImage was found on this machine\n\n" + notFoundHelp())
	}
	root, err := extractAppImage(appImage, func(msg string, a ...any) { info(msg, a...) })
	if err != nil {
		return "", err
	}
	if err := appImageShortcut(root); err != nil {
		info("Could not write the menu shortcut: %v", err)
	}
	return root, nil
}

// afterInstallNote tells the user how to start the copy the panel was installed
// into, because on Linux that is not the AppImage they launched.
func afterInstallNote() string {
	if !appImageReady() {
		return ""
	}
	root := extractedRoot()
	return "  Start Freebuff from the extracted copy, not the AppImage:\n" +
		"      " + colBold + filepath.Join(root, "freebuff") + colReset + "\n" +
		"  A " + colBold + "Freebuff" + colReset + " entry was added to your application menu,\n" +
		"  which starts that same copy. Using the AppImage directly would\n" +
		"  mount it read-only again, and the panel would not be there."
}
