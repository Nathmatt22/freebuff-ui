package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestInstall builds a throwaway install tree that looks enough like
// Freebuff for inject() to accept it.
func newTestInstall(t *testing.T, indexHTML string) (install, ui string) {
	t.Helper()
	install = t.TempDir()
	ui = filepath.Join(install, "resources", "orchestrator", "ui")
	if err := os.MkdirAll(filepath.Join(ui, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(indexPath(ui), []byte(indexHTML), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	return install, ui
}

const sampleIndex = `<!doctype html>
<html>
  <head><title>Freebuff</title></head>
  <body>
    <div id="root"></div>
    <div id="startup-recovery"></div>
  </body>
</html>
`

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestInjectIsIdempotent(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)

	if err := inject(ui, true); err != nil {
		t.Fatalf("first inject: %v", err)
	}
	first := readFile(t, indexPath(ui))

	if err := inject(ui, true); err != nil {
		t.Fatalf("second inject: %v", err)
	}
	second := readFile(t, indexPath(ui))

	if first != second {
		t.Fatalf("inject is not idempotent:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if strings.Count(second, markerStart) != 1 || strings.Count(second, markerEnd) != 1 {
		t.Fatalf("expected exactly one marker pair, got %d/%d", strings.Count(second, markerStart), strings.Count(second, markerEnd))
	}
	if !strings.Contains(second, engineName) {
		t.Fatalf("engine script tag missing")
	}
	if _, err := os.Stat(backupPath(ui)); err != nil {
		t.Fatalf("backup not written: %v", err)
	}
}

func TestMarkerBoundsRejectsMalformed(t *testing.T) {
	cases := map[string]string{
		"duplicate start": markerStart + "\n" + markerStart + "\n" + markerEnd,
		"missing end":     "x" + markerStart + "y",
		"out of order":    markerEnd + "x" + markerStart,
	}
	for name, html := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := markerBounds(html); err == nil {
				t.Fatalf("expected an error for %s", name)
			}
		})
	}

	if _, _, found, err := markerBounds("plain html"); err != nil || found {
		t.Fatalf("plain html should report no markers, got found=%v err=%v", found, err)
	}
}

func TestUninstallRestoresPristineBackup(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)
	if err := inject(ui, true); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := uninstall(ui, true); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if got := readFile(t, indexPath(ui)); got != sampleIndex {
		t.Fatalf("uninstall did not restore the original index:\n%s", got)
	}
	if _, err := os.Stat(backupPath(ui)); !os.IsNotExist(err) {
		t.Fatalf("backup should be consumed on a clean uninstall")
	}
}

// A Freebuff update rewrites index.html after we injected. Uninstall must
// remove our block but must not put the old backup back over the new app.
func TestUninstallKeepsNewerIndex(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)
	if err := inject(ui, true); err != nil {
		t.Fatalf("inject: %v", err)
	}

	updated := readFile(t, indexPath(ui)) + "\n<!-- freebuff 1.4.0 updated this file -->\n"
	if err := os.WriteFile(indexPath(ui), []byte(updated), 0o644); err != nil {
		t.Fatalf("simulate app update: %v", err)
	}

	if err := uninstall(ui, true); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	got := readFile(t, indexPath(ui))
	if strings.Contains(got, markerStart) {
		t.Fatalf("injection marker survived uninstall:\n%s", got)
	}
	if !strings.Contains(got, "freebuff 1.4.0 updated this file") {
		t.Fatalf("uninstall overwrote the newer index:\n%s", got)
	}
	if _, err := os.Stat(backupPath(ui)); err != nil {
		t.Fatalf("stale backup should be kept for the user, not silently discarded: %v", err)
	}
}

func TestLicenseIndexRefused(t *testing.T) {
	_, ui := newTestInstall(t, "<html><body>not the app</body></html>")
	if err := inject(ui, true); err == nil {
		t.Fatalf("inject should refuse a page that is not the Freebuff UI")
	}
	if _, err := os.Stat(backupPath(ui)); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written when the page is refused")
	}
}

func TestBakeThemeUnwrapsEnvelopeAndRejectsGarbage(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)

	envelope := filepath.Join(t.TempDir(), "theme.fbtheme")
	if err := os.WriteFile(envelope, []byte(`{"format":"fbtheme","formatVersion":1,"theme":{"v":1,"name":"Nord"}}`), 0o644); err != nil {
		t.Fatalf("write theme: %v", err)
	}
	if err := bakeTheme(ui, envelope); err != nil {
		t.Fatalf("bakeTheme: %v", err)
	}
	baked := readFile(t, filepath.Join(assetsDir(ui), defaultName))
	if !strings.Contains(baked, `window.__FREEBUFF_THEME_DEFAULT__`) || !strings.Contains(baked, `"Nord"`) {
		t.Fatalf("baked theme is not the unwrapped theme object:\n%s", baked)
	}
	if strings.Contains(baked, `"format":"fbtheme"`) {
		t.Fatalf("baked theme still carries the envelope")
	}

	for name, body := range map[string]string{
		"empty":    "",
		"array":    "[1,2,3]",
		"null":     "null",
		"envelope": `{"theme":42}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "bad.json")
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			if err := bakeTheme(ui, p); err == nil {
				t.Fatalf("expected bakeTheme to reject %s", name)
			}
		})
	}
}

func TestAtomicWriteReplacesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := atomicWriteFile(path, []byte("first"), 0o644); err != nil {
		t.Fatalf("write 1: %v", err)
	}
	if err := atomicWriteFile(path, []byte("second"), 0o644); err != nil {
		t.Fatalf("write 2 (replace): %v", err)
	}
	if got := readFile(t, path); got != "second" {
		t.Fatalf("got %q, want %q", got, "second")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".fbts-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestManifestChecksumMatchesBackup(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)
	if err := inject(ui, true); err != nil {
		t.Fatalf("inject: %v", err)
	}
	m := readManifest(ui)
	if m == nil {
		t.Fatalf("manifest missing")
	}
	backup := readFile(t, backupPath(ui))
	if m.OriginalSHA != sha256Hex([]byte(backup)) {
		t.Fatalf("manifest checksum does not describe the backup")
	}
	if !m.HadBackup {
		t.Fatalf("manifest should record the backup")
	}
	b, _ := json.Marshal(m)
	if !strings.Contains(string(b), "originalIndexSha256") {
		t.Fatalf("manifest JSON lost its checksum field")
	}
}
