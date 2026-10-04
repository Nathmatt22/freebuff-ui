//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// t.Setenv scopes XDG_DATA_HOME to a temp dir so these tests never touch the
// real state directory of whoever is running them.
func withStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	return dir
}

func TestAppStateDirFollowsXDG(t *testing.T) {
	withStateDir(t)
	got := appStateDir()
	want := filepath.Join(os.Getenv("XDG_DATA_HOME"), "FreebuffThemeStudio")
	if got != want {
		t.Fatalf("appStateDir() = %q, want %q", got, want)
	}
}

func TestExtractedRootIsUnderStateDir(t *testing.T) {
	withStateDir(t)
	if !strings.HasPrefix(extractedRoot(), appStateDir()) {
		t.Fatalf("extractedRoot %q must live under %q", extractedRoot(), appStateDir())
	}
}

// A machine with no AppImage must not be pushed into an extraction attempt, or
// findInstallDir would run for a long time and then fail.
func TestNoAppImageMeansNoAutoSetup(t *testing.T) {
	withStateDir(t)
	t.Setenv("HOME", t.TempDir())
	if appImageFile() != "" {
		t.Fatalf("an empty home must not yield an AppImage, got %q", appImageFile())
	}
	if appImageReady() {
		t.Fatalf("nothing extracted yet, so appImageReady must be false")
	}
	if looksLikeAppImage() {
		t.Fatalf("with no AppImage present, looksLikeAppImage must be false")
	}
}

// The menu shortcut is what keeps the user from launching the AppImage again and
// silently losing the panel, so its Exec line has to point at the extracted
// binary rather than the AppImage.
func TestAppImageShortcutPointsAtExtractedCopy(t *testing.T) {
	base := withStateDir(t)
	root := filepath.Join(base, "FreebuffThemeStudio", "squashfs-root")
	ui := filepath.Join(root, "resources", "orchestrator", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(indexPath(ui), []byte(sampleIndex), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	// A launcher the shortcut will find.
	exe := filepath.Join(root, "freebuff")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}

	if err := appImageShortcut(root); err != nil {
		t.Fatalf("appImageShortcut: %v", err)
	}
	p := filepath.Join(base, "applications", "freebuff-themed.desktop")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read shortcut: %v", err)
	}
	body := string(b)
	if !strings.Contains(body, "Exec="+exe) {
		t.Fatalf("the shortcut must exec the extracted binary, got:\n%s", body)
	}
	if strings.Contains(body, ".AppImage") {
		t.Fatalf("the shortcut must not point back at the AppImage:\n%s", body)
	}
	if !strings.Contains(body, "[Desktop Entry]") || !strings.Contains(body, "Type=Application") {
		t.Fatalf("not a valid desktop entry:\n%s", body)
	}
}

// The post-install hint is the only place the user learns which copy to start,
// so it has to name the extracted path.
func TestAfterInstallNoteNamesExtractedCopy(t *testing.T) {
	withStateDir(t)
	if note := afterInstallNote(); note != "" {
		t.Fatalf("nothing is extracted yet, so the note must be empty, got:\n%s", note)
	}
	root := filepath.Join(appStateDir(), "squashfs-root")
	ui := filepath.Join(root, "resources", "orchestrator", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(indexPath(ui), []byte(sampleIndex), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	note := afterInstallNote()
	if !strings.Contains(note, "squashfs-root") {
		t.Fatalf("the note must name the extracted copy, got:\n%s", note)
	}
	if !strings.Contains(note, "AppImage") {
		t.Fatalf("the note should say why the AppImage is not the right one to start:\n%s", note)
	}
}

// Once the panel is injected, the copy is ready; if it is not injected, running
// the AppImage would give a Freebuff without the panel.
func TestAppImageStaleDetectsMissingInjection(t *testing.T) {
	withStateDir(t)
	root := filepath.Join(appStateDir(), "squashfs-root")
	ui := filepath.Join(root, "resources", "orchestrator", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(indexPath(ui), []byte(sampleIndex), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if !appImageReady() {
		t.Fatalf("an extracted install must be reported ready")
	}
	if !appImageStale() {
		t.Fatalf("an extracted copy without the panel must be reported stale")
	}
	if err := inject(ui, true); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if appImageStale() {
		t.Fatalf("an injected copy must not be reported stale")
	}
}

// autoSetupAppImage must be a no-op when there is no AppImage, and must not
// invent an install out of nothing.
func TestAutoSetupWithoutAppImageFails(t *testing.T) {
	withStateDir(t)
	t.Setenv("HOME", t.TempDir())
	if _, err := autoSetupAppImage(); err == nil {
		t.Fatalf("auto setup without an AppImage must report an error")
	} else if !strings.Contains(err.Error(), "AppImage") {
		t.Fatalf("the error should name the AppImage, got: %v", err)
	}
}

// isExecFormat is what turns a non-executable download into advice instead of a
// bare "permission denied".
func TestExecFormatDetection(t *testing.T) {
	for _, out := range []string{
		"bash: ./Freebuff.AppImage: cannot execute binary file",
		"cannot exec: Exec format error",
		"sh: permission denied",
	} {
		if !isExecFormat(out) {
			t.Fatalf("should have recognised %q", out)
		}
	}
	if isExecFormat("all good") {
		t.Fatalf("unrelated output must not be treated as an exec problem")
	}
}

// staleCopyWarning is the guard against "I installed it and nothing happened".
// It must stay quiet when the running copy is the patched one, and speak up
// when it is not.
func TestStaleCopyWarningQuietWhenRunningCopyIsPatched(t *testing.T) {
	withStateDir(t)
	root := filepath.Join(appStateDir(), "squashfs-root")
	ui := filepath.Join(root, "resources", "orchestrator", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(indexPath(ui), []byte(sampleIndex), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := inject(ui, true); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if appImageStale() {
		t.Fatalf("just injected, must not be stale")
	}
	// The ui is the patched tree itself, so there is nothing to warn about.
	if w := staleCopyWarning(ui); w != "" {
		t.Fatalf("the patched copy needs no warning, got:\n%s", w)
	}
}

func TestStaleCopyWarningQuietWhenNothingExtracted(t *testing.T) {
	withStateDir(t)
	t.Setenv("HOME", t.TempDir())
	if w := staleCopyWarning(filepath.Join(t.TempDir(), "ui")); w != "" {
		t.Fatalf("with no AppImage set up there is nothing to warn about, got:\n%s", w)
	}
}

/*
 * The report from a real machine: the injector printed "Freebuff is already
 * running" on a machine where nothing was running. It had matched itself, since
 * this program is called FreebuffThemeInjector and its name contains the app's
 * name. These tests pin that down.
 */
func TestInjectorIsNotMistakenForTheApp(t *testing.T) {
	if isFreebuffProcess(strconv.Itoa(os.Getpid())) {
		t.Fatalf("the injector must never report itself as the running app")
	}
}

// An AppImage launcher is versioned, so the name cannot be guessed. The
// .desktop file it ships is the reliable source.
func TestRelaunchExeUsesDesktopEntry(t *testing.T) {
	root := t.TempDir()
	// A launcher with a name no hardcoded list would have contained.
	exe := filepath.Join(root, "freebuff-desktop-0.0.158")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}
	desktop := "[Desktop Entry]\nType=Application\nName=Freebuff\nExec=" +
		"freebuff-desktop-0.0.158 %U\nTerminal=true\n"
	if err := os.WriteFile(filepath.Join(root, "freebuff.desktop"), []byte(desktop), 0o644); err != nil {
		t.Fatalf("write desktop: %v", err)
	}
	got := relaunchExe(root)
	if got == "" {
		t.Fatalf("the launcher must be found through the .desktop file")
	}
	if filepath.Base(got) != "freebuff-desktop-0.0.158" {
		t.Fatalf("relaunchExe() = %q, want the versioned launcher", got)
	}
}

func TestExecFieldFromDesktop(t *testing.T) {
	cases := map[string]string{
		"Exec=freebuff\n":                    "freebuff",
		"Exec=freebuff %U\n":                 "freebuff",
		"Exec=/opt/freebuff --flag %F\n":     "/opt/freebuff",
		"Exec=\"/opt/my app/freebuff\" %U\n": "/opt/my app/freebuff",
		"Name=Freebuff\n":                    "",
		"Exec=%UNKNOWN%\n":                   "",
		"Exec=\n":                            "",
	}
	for body, want := range cases {
		if got := execFieldFromDesktop(body); got != want {
			t.Fatalf("execFieldFromDesktop(%q) = %q, want %q", body, got, want)
		}
	}
}

// A versioned .desktop entry is what the real AppImage ships, so the shortcut
// has to be written from it rather than skipped.
func TestAppImageShortcutUsesDesktopEntryLauncher(t *testing.T) {
	base := withStateDir(t)
	root := filepath.Join(base, "FreebuffThemeStudio", "squashfs-root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	exe := filepath.Join(root, "freebuff-desktop-0.0.158")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "freebuff.desktop"),
		[]byte("[Desktop Entry]\nExec=freebuff-desktop-0.0.158 %U\n"), 0o644); err != nil {
		t.Fatalf("write desktop: %v", err)
	}
	if err := appImageShortcut(root); err != nil {
		t.Fatalf("appImageShortcut must succeed now: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(base, "applications", "freebuff-themed.desktop"))
	if err != nil {
		t.Fatalf("read shortcut: %v", err)
	}
	if !strings.Contains(string(b), "Exec="+exe) {
		t.Fatalf("the shortcut must point at the versioned launcher, got:\n%s", string(b))
	}
}
