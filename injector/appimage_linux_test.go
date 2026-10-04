//go:build linux

package main

import (
	"os"
	"path/filepath"
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
