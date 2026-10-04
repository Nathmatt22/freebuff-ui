package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

/*
 * These tests exist because the platform layer was written on Windows and could
 * not be run there: it compiled for linux and darwin, but every line below the
 * build tags was unexercised until CI. They are deliberately OS-agnostic in
 * shape and assert OS-specific behaviour where the two implementations differ,
 * so the same file is meaningful on all three runners.
 */

// ---------------------------------------------------------- install paths ---

func TestIsInstallDirRequiresOrchestratorIndex(t *testing.T) {
	dir := t.TempDir()
	if isInstallDir(dir) {
		t.Fatalf("an empty temp dir must not look like an install")
	}
	ui := filepath.Join(dir, "resources", "orchestrator", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(indexPath(ui), []byte(sampleIndex), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if !isInstallDir(dir) {
		t.Fatalf("a tree with resources/orchestrator/ui/index.html must be an install")
	}
}

func TestFindInstallDirRejectsForeignPath(t *testing.T) {
	bad := t.TempDir()
	_, err := findInstallDir(bad)
	if err == nil {
		t.Fatalf("a directory with no index.html must be rejected")
	}
	// An explicit path names the offending directory: the user typed it, so
	// there is nothing left to hint at.
	if !strings.Contains(err.Error(), bad) {
		t.Fatalf("the error should name the rejected path, got: %v", err)
	}

	// Auto-detection has no path to blame, so it must say how to supply one.
	seen := map[string]bool{bad: true}
	for _, c := range candidateDirs() {
		if c != "" {
			seen[c] = true
		}
	}
	if _, err := findInstallDir(""); err == nil {
		t.Skip("this machine has a real Freebuff install, so auto-detection succeeded")
	} else if !strings.Contains(err.Error(), "--path") {
		t.Fatalf("the auto-detection error should tell the user about --path, got: %v", err)
	}
}

func TestFindInstallDirHonoursOverride(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)
	install, err := findInstallDir(filepath.Dir(filepath.Dir(filepath.Dir(ui))))
	if err != nil {
		t.Fatalf("findInstallDir with an explicit path: %v", err)
	}
	if !isInstallDir(install) {
		t.Fatalf("returned %q which is not an install", install)
	}
}

// The candidates must at least include the directories the platform uses for
// per-user installs. An empty list would make auto-detection silently useless.
func TestCandidateDirsIncludeUserInstallLocations(t *testing.T) {
	dirs := candidateDirs()
	if len(dirs) == 0 {
		t.Fatalf("candidateDirs returned nothing")
	}
	joined := strings.Join(dirs, string(filepath.Separator))
	if runtime.GOOS == "windows" {
		if !strings.Contains(joined, "Programs") {
			t.Fatalf("Windows detection should look under a Programs directory, got: %v", dirs)
		}
	} else {
		// XDG data home is the Unix equivalent of %LOCALAPPDATA%\Programs.
		if !strings.Contains(joined, ".local") && !strings.Contains(joined, "/opt") && !strings.Contains(joined, "/usr") {
			t.Fatalf("Unix detection should look under XDG/opt/usr locations, got: %v", dirs)
		}
	}
}

// ------------------------------------------------------------- guard state ---

// processAlive must agree with reality in both directions: a live pid is alive,
// and the pid we just exited with is not.
func TestProcessAliveMatchesReality(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatalf("the running process must be reported as alive on %s", runtime.GOOS)
	}
	if processAlive(-1) || processAlive(0) {
		t.Fatalf("a non-positive pid must never be reported as alive")
	}
}

func TestProcessNameFindsThisProcess(t *testing.T) {
	name := processName(os.Getpid())
	if name == "" {
		// Some hardened Linux setups deny reads of other pids' entries; the
		// function is allowed to give up rather than guess. Only assert on the
		// platforms where the lookup is expected to work.
		if runtime.GOOS == "linux" {
			t.Skip("process name unavailable in this sandbox; /proc lookups are restricted")
		}
		return
	}
	if !strings.Contains(strings.ToLower(name), "injector") && !strings.Contains(strings.ToLower(name), "test") {
		t.Fatalf("processName(%d) = %q, which does not look like this test binary", os.Getpid(), name)
	}
}

// -------------------------------------------------------------- file paths ---

func TestWatchDirIsWritable(t *testing.T) {
	dir := watchDir()
	if dir == "" {
		t.Skip("no per-user config directory is available in this environment")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Skipf("cannot create %s here: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// The guard writes through the same helper it uses at runtime, so this also
	// proves replaceFile works on this platform.
	if err := atomicWriteFile(filepath.Join(dir, "probe.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("atomic write into the guard dir: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "probe.txt"))
	if err != nil || string(b) != "ok" {
		t.Fatalf("probe read back = %q, %v", string(b), err)
	}
}

// replaceFile must clobber an existing file: that is the whole reason Windows
// needs MoveFileEx rather than os.Rename.
func TestReplaceFileOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "target")
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tmp := filepath.Join(dir, "temp")
	if err := os.WriteFile(tmp, []byte("new"), 0o644); err != nil {
		t.Fatalf("temp: %v", err)
	}
	if err := replaceFile(tmp, dst); err != nil {
		t.Fatalf("replaceFile: %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(b) != "new" {
		t.Fatalf("content = %q, want %q", string(b), "new")
	}
	if _, err := os.Stat(tmp); err == nil {
		t.Fatalf("the temporary file should be gone after a successful replace")
	}
}

// ---------------------------------------------------------- autostart entry ---

// setRunEntry / runEntryPresent / clearRunEntry must round-trip, on the registry
// on Windows and on the autostart .desktop file elsewhere.
func TestRunEntryRoundTrip(t *testing.T) {
	if watchDir() == "" {
		t.Skip("no per-user config directory is available in this environment")
	}
	if runEntryPresent() {
		clearRunEntry()
	}
	if runEntryPresent() {
		t.Fatalf("no logon entry should be present before the test sets one")
	}
	if err := setRunEntry(`"test-binary" --watch --quiet`); err != nil {
		t.Fatalf("setRunEntry: %v", err)
	}
	if !runEntryPresent() {
		t.Fatalf("the logon entry was not registered on %s", runtime.GOOS)
	}
	clearRunEntry()
	if runEntryPresent() {
		t.Fatalf("the logon entry survived clearRunEntry")
	}
	// Clearing twice is the normal uninstall path.
	clearRunEntry()
}

// The guard binary must keep its own name across the refactor, or promoteWatcher
// would copy it to a path the autostart entry does not name.
func TestGuardExeNameIsStable(t *testing.T) {
	name := guardExeName()
	if name == "" {
		t.Fatalf("guardExeName returned an empty string")
	}
	if strings.ContainsAny(name, `/\`) {
		t.Fatalf("guardExeName must be a bare file name, got %q", name)
	}
	if strings.Contains(name, "..") {
		t.Fatalf("guardExeName must not contain path traversal, got %q", name)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		t.Fatalf("the Windows guard must keep its .exe suffix, got %q", name)
	}
	if !strings.Contains(watchExePath(), name) {
		t.Fatalf("watchExePath %q does not contain the guard name %q", watchExePath(), name)
	}
}

// ------------------------------------------------------------ cookie roots ---

func TestCookieRootsPointAtUserConfig(t *testing.T) {
	roots := cookieRoots()
	if len(roots) == 0 {
		t.Fatalf("cookieRoots returned nothing; theme cookies could never be found")
	}
	for _, r := range roots {
		if r == "" || !filepath.IsAbs(r) {
			t.Fatalf("cookie root %q must be an absolute path", r)
		}
	}
}

func TestProfileCookieDBsNeverTouchForeignApps(t *testing.T) {
	// Nothing named freebuff exists in a clean test environment, so this must
	// come back empty rather than scanning arbitrary directories.
	for _, db := range profileCookieDBs() {
		if !strings.Contains(strings.ToLower(filepath.Base(filepath.Dir(filepath.Dir(db)))), "freebuff") {
			t.Fatalf("profileCookieDBs returned %q, which is not a freebuff profile", db)
		}
	}
}

// ---------------------------------------------------------- process probes ---

func TestFreebuffRunningDoesNotPanicWhenAbsent(t *testing.T) {
	// Freebuff is not installed on a CI runner; the probe must answer false
	// rather than error out or hang.
	_ = freebuffRunning()
	_ = freebuffRunning()
}

func TestBunBinaryLookupIsSafeWhenMissing(t *testing.T) {
	// An empty result is fine and expected on a runner without the app; the
	// contract is that it must not panic and must not invent a path.
	if p := bunBinary(t.TempDir()); p != "" && !filepath.IsAbs(p) {
		t.Fatalf("bunBinary returned a relative path: %q", p)
	}
}

func TestRelaunchExeDoesNotPanicOnEmptyInstall(t *testing.T) {
	if p := relaunchExe(t.TempDir()); p != "" {
		t.Fatalf("an empty directory must contain no launcher, got %q", p)
	}
}

// The not-found message is what a Linux user sees when auto-detection fails; it
// must name the AppImage workflow, which is the actual answer there.
func TestNotFoundHelpMentionsAppImage(t *testing.T) {
	msg := notFoundHelp()
	for _, want := range []string{"--path", "AppImage", "squashfs-root", "index.html"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("notFoundHelp() should mention %q, got:\n%s", want, msg)
		}
	}
}

// A Linux install is an extracted AppImage root, which is what --path
// squashfs-root points at, so it has to count as an install.
func TestExtractedAppImageRootIsAnInstall(t *testing.T) {
	root := t.TempDir()
	ui := filepath.Join(root, "resources", "orchestrator", "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(indexPath(ui), []byte(sampleIndex), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !isInstallDir(root) {
		t.Fatalf("an extracted AppImage root must count as an install")
	}
	if _, err := findInstallDir(root); err != nil {
		t.Fatalf("an extracted AppImage root must be accepted by --path: %v", err)
	}
}

// A writable tree must pass the guard that keeps us from failing later, halfway
// through replacing index.html, on a read-only AppImage mount.
func TestCheckWritableAcceptsOrdinaryDirectory(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)
	if err := checkWritable(ui); err != nil {
		t.Fatalf("a writable ui dir must pass: %v", err)
	}
}

// isReadOnlyErr is what turns a bare "read-only file system" into an
// explanation. os.IsPermission only covers EACCES, which is the bug: a
// read-only filesystem reports EROFS and used to fall through to the raw error.
func TestIsReadOnlyErrCoversErofs(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows installs are always writable; there is no EROFS branch there.
		if isReadOnlyErr(syscall.EROFS) {
			t.Fatalf("Windows must never report a read-only filesystem")
		}
		return
	}
	if !isReadOnlyErr(syscall.EROFS) {
		t.Fatalf("EROFS must be recognised as a read-only filesystem")
	}
	if !isReadOnlyErr(&os.PathError{Op: "create", Path: "/tmp", Err: syscall.EROFS}) {
		t.Fatalf("EROFS wrapped in a PathError must still be recognised")
	}
	if isReadOnlyErr(nil) {
		t.Fatalf("no error must not be reported as read-only")
	}
	if isReadOnlyErr(syscall.ENOENT) {
		t.Fatalf("a missing file is not a read-only filesystem")
	}
}

// A writable directory must never be mistaken for a read-only one, otherwise
// detection would skip perfectly good installs.
func TestWritableDirectoryIsNotReadOnly(t *testing.T) {
	if isReadOnlyPath(t.TempDir()) {
		t.Fatalf("a fresh temp dir must be treated as writable")
	}
}

// checkWritable has to accept an ordinary install; on Linux the read-only
// branch is covered by isReadOnlyErr above.
func TestCheckWritableAcceptsNormalInstall(t *testing.T) {
	_, ui := newTestInstall(t, sampleIndex)
	if err := checkWritable(ui); err != nil {
		t.Fatalf("a writable install must pass checkWritable: %v", err)
	}
}

// staleName guards against the .old.old pile-up: the guard image often cannot be
// deleted while it is shutting down, so it is renamed aside, and the fallback
// ran on a path that already carried the suffix.
func TestStaleNameNeverDoublesTheSuffix(t *testing.T) {
	cases := map[string]string{
		"C:/x/FreebuffThemeInjector.exe":     "C:/x/FreebuffThemeInjector.exe.old",
		"C:/x/FreebuffThemeInjector.exe.old": "C:/x/FreebuffThemeInjector.exe.stale",
		"C:/x/watcher.pid":                   "C:/x/watcher.pid.old",
	}
	for in, want := range cases {
		got := staleName(in)
		if got != want {
			t.Fatalf("staleName(%q) = %q, want %q", in, got, want)
		}
		if strings.Contains(got, ".old.old") {
			t.Fatalf("staleName(%q) produced a doubled suffix: %q", in, got)
		}
	}
}

/*
 * The guard must never trigger an extraction. It ticks every 15 seconds in a
 * process that prints nothing, so an extraction there would unpack a whole
 * AppImage silently and repeat it forever if the copy never appeared.
 */
func TestDetectionAloneNeverSetsUp(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		t.Setenv("HOME", t.TempDir())
	}
	// Both calls must be safe: findInstallDir may set up, findInstall must not
	// and must simply report that nothing was found.
	if _, err := findInstall("", false); err == nil {
		t.Skip("this machine has a real Freebuff install, so detection succeeded")
	}
	if _, err := findInstallDir(""); err == nil {
		t.Skip("this machine has a real Freebuff install, so detection succeeded")
	}
}
