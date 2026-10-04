//go:build !windows

// Freebuff Theme Studio - platform layer for Linux and macOS.
//
// Everything in main.go that is specific to one operating system lives here or
// in platform_windows.go, behind identical function names. The injector itself
// is portable Go; only these few dozen lines differ.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ---------------------------------------------------------------- console ---

// Terminals on Unix are ANSI to begin with, and Go writes UTF-8 directly.
func enableVT() bool { return true }

// ------------------------------------------------------------------ files ---

// os.Rename is atomic and replaces the destination outright on Linux and macOS,
// so there is nothing Windows' MoveFileEx does that we need here.
func replaceFile(temp, path string) error {
	return os.Rename(temp, path)
}

// ------------------------------------------------------------- processes ---

// hideProc keeps helper commands (powershell-free helpers, bun) from flashing a
// console window. On Unix they inherit ours and nothing is shown anyway.
func hideProc(cmd *exec.Cmd) {}

// detachProc starts a command in its own session so it survives our exit.
func detachProc(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// --------------------------------------------------------------- locating ---

/*
 * Installs are found the same way they are installed: per-user under XDG, then
 * system-wide. Unlike Windows there is no single "Programs" convention, so the
 * common layouts are enumerated explicitly.
 */
func candidateDirs() []string {
	var out []string

	if exe := runningExecutable(); exe != "" {
		out = append(out, filepath.Dir(exe))
	}

	names := []string{
		"@codebufffreebuff-desktop",
		"freebuff-desktop",
		"Freebuff",
		"Freebuff Desktop",
		"freebuff",
	}

	roots := []string{
		os.Getenv("XDG_DATA_HOME"),
		os.Getenv("XDG_HOME"),
		os.Getenv("SNAP"),
		"/usr/share",
		"/usr/lib",
		"/usr/local/share",
		"/usr/local/lib",
		"/opt",
		"/Applications",
		"/Applications/Users",
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append([]string{
			filepath.Join(home, ".local", "share"),
			filepath.Join(home, ".local", "opt"),
			filepath.Join(home, "opt"),
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".var", "app"),
		}, roots...)
	}

	for _, root := range roots {
		if root == "" {
			continue
		}
		for _, n := range names {
			out = append(out, filepath.Join(root, n))
			// Snap's layout inserts the app name twice.
			out = append(out, filepath.Join(root, n, "current", n))
		}
	}

	// A running AppImage: the only copy of the files that exists on disk.
	out = append(out, appImageMounts()...)

	// An extracted AppImage, either squashfs-root next to the download or an
	// AppImageLauncher directory, which is writable - unlike the live mount.
	if home, err := os.UserHomeDir(); err == nil {
		for _, base := range []string{"", filepath.Join(".local", "share"), filepath.Join("Applications")} {
			root := filepath.Join(home, base)
			out = append(out,
				filepath.Join(root, "squashfs-root"),
				filepath.Join(root, "Applications", "Freebuff"),
				filepath.Join(root, "Freebuff-linux-x86_64.AppImage"),
				filepath.Join(root, "Freebuff-linux-arm64.AppImage"),
			)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		out = append(out, wd)
	}
	return out
}

// runningExecutable prefers the live process, because that is the install the
// user actually runs. /proc holds the real path for every process we can see.
/*
 * An AppImage is a single file that the kernel mounts read-only under
 * /tmp/.mount_XXXXXX while it runs, so a running Freebuff is found by its
 * process rather than by an install directory: /proc/<pid>/exe resolves to the
 * binary inside that mount, and the install is its parent directory.
 *
 * The process name is checked as well as the exe path, because an AppImage can
 * be renamed to anything the user likes while the process keeps a generic name.
 */
func runningExecutable() string {
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return ""
	}
	for _, p := range procs {
		if !p.IsDir() || !isNumericName(p.Name()) {
			continue
		}
		comm := strings.ToLower(processNameFromProc(p.Name()))
		looksLikeUs := strings.Contains(comm, "freebuff") || strings.Contains(comm, "codebuff")
		if !looksLikeUs {
			continue
		}
		exe, err := os.Readlink(filepath.Join("/proc", p.Name(), "exe"))
		if err != nil {
			continue
		}
		// "<mount>/usr/bin/freebuff" -> "<mount>", which is the install root.
		for dir := filepath.Dir(exe); dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			if isInstallDir(dir) {
				return exe
			}
		}
	}
	return ""
}

// isReadOnlyErr recognises the two ways a filesystem refuses a write. EROFS
// ("read-only file system") is what an AppImage mount answers with, and it is
// not covered by os.IsPermission, which only sees EACCES.
func isReadOnlyErr(err error) bool {
	return errors.Is(err, syscall.EROFS) || errors.Is(err, syscall.EACCES) || os.IsPermission(err)
}

// isReadOnlyPath reports whether a directory cannot be written to, checked
// before anything is touched so the user gets the reason instead of a failure
// partway through replacing index.html.
func isReadOnlyPath(dir string) bool {
	// Ask the filesystem directly: this is the same call that would fail later.
	probe, err := os.CreateTemp(dir, ".fbts-probe-*")
	if err != nil {
		if isReadOnlyErr(err) {
			return true
		}
		// Any other failure here (missing dir, for instance) is not proof of a
		// read-only filesystem.
		return false
	}
	name := probe.Name()
	probe.Close()
	_ = os.Remove(name)

	// A writable squashfs mount is unusual, but treat one as read-only anyway:
	// the AppImage would be replaced on next launch and take the panel with it.
	mounts, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(mounts), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || !strings.HasPrefix(f[2], "squashfs") {
			continue
		}
		point := strings.TrimSuffix(f[1], "/")
		if dir == point || strings.HasPrefix(dir, point+"/") {
			return true
		}
	}
	return false
}

// appImageMounts lists the squashfs mounts an AppImage creates while running.
// They are the only place the app's files exist on disk for a portable build.
func appImageMounts() []string {
	var out []string
	entries, err := os.ReadDir("/tmp")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || !strings.HasPrefix(name, ".mount_") {
			continue
		}
		out = append(out, filepath.Join("/tmp", name))
	}
	return out
}

func isNumericName(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

// freebuffRunning asks the process table; pgrep is not guaranteed to exist, so
// /proc is walked directly.
func freebuffRunning() bool {
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, p := range procs {
		if !p.IsDir() || !isNumericName(p.Name()) {
			continue
		}
		if strings.Contains(strings.ToLower(processNameFromProc(p.Name())), "freebuff") {
			return true
		}
	}
	return false
}

// stopFreebuff closes the app and waits, because Chromium holds the cookie jar
// in memory and would write it back over any changes we make. A polite TERM
// first so it can shut down cleanly; SIGKILL only if it will not go.
func stopFreebuff(quiet bool) bool {
	if !freebuffRunning() {
		return true
	}
	if !quiet {
		info("Asking Freebuff to close…")
	}
	signalAll("freebuff", syscall.SIGTERM)
	for i := 0; i < 60; i++ {
		if !freebuffRunning() {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	signalAll("freebuff", syscall.SIGKILL)
	for i := 0; i < 40; i++ {
		if !freebuffRunning() {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !quiet {
		warn("Freebuff is still running - close it by hand and try again")
	}
	return false
}

func signalAll(match string, sig syscall.Signal) {
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	for _, p := range procs {
		if !p.IsDir() || !isNumericName(p.Name()) {
			continue
		}
		if !strings.Contains(strings.ToLower(processNameFromProc(p.Name())), match) {
			continue
		}
		pid, err := strconv.Atoi(p.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Signal(sig)
		}
	}
}

/*
 * relaunchExe finds the Freebuff launcher for an install. Unlike Windows there
 * is no fixed name, so the usual shapes are tried in turn.
 *
 * An AppImage install is a squashfs mount: the real binary lives under usr/bin
 * inside it, and running that copy directly would start a second, unmounted
 * instance. So a live mount has no relaunch target and the user is told to start
 * the app themselves, which is also what the update path needs.
 */
func relaunchExe(install string) string {
	names := []string{
		"Freebuff", "freebuff", "Freebuff.sh", "freebuff.sh",
		"Freebuff.AppImage", "freebuff.AppImage",
		"freebuff-desktop", "Freebuff Desktop",
		"bin/Freebuff", "bin/freebuff",
		"usr/bin/Freebuff", "usr/bin/freebuff",
	}
	for _, n := range names {
		p := filepath.Join(install, n)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

func relaunch(install string) {
	exe := relaunchExe(install)
	if exe == "" {
		warn("Could not find a Freebuff launcher in %s to relaunch", install)
		return
	}
	info("Launching Freebuff…")
	cmd := exec.Command(exe)
	cmd.Dir = install
	hideProc(cmd)
	if err := cmd.Start(); err != nil {
		warn("Could not launch Freebuff: %v", err)
		return
	}
	go cmd.Wait()
}

// openFolder opens a directory in the desktop file manager.
func openFolder(dir string) {
	for _, opener := range []string{"xdg-open", "gio", "nautilus", "dolphin", "thunar", "open"} {
		if _, err := exec.LookPath(opener); err != nil {
			continue
		}
		var cmd *exec.Cmd
		if opener == "gio" {
			cmd = exec.Command(opener, "open", dir)
		} else {
			cmd = exec.Command(opener, dir)
		}
		hideProc(cmd)
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return
		}
	}
	warn("No file manager found; open %s by hand", dir)
}

// ------------------------------------------------------------------ watch ---

// hideConsoleWindow has nothing to do: the guard is started detached, so it
// never gets a terminal of its own.
func hideConsoleWindow() {}

// processAlive checks with signal 0, which performs the permission and
// existence checks without actually signalling anything.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	_ = p.Release()
	return err == nil
}

func processName(pid int) string { return processNameFromProc(strconv.Itoa(pid)) }

func processNameFromProc(pid string) string {
	if b, err := os.ReadFile(filepath.Join("/proc", pid, "comm")); err == nil {
		return strings.TrimSpace(string(b))
	}
	// No comm (older kernels): the first argv entry is the next best thing.
	b, err := os.ReadFile(filepath.Join("/proc", pid, "cmdline"))
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.SplitN(string(b), "\x00", 2)[0])
	return filepath.Base(line)
}

// watchBaseDir is the per-user config dir, the Unix answer to LOCALAPPDATA.
func watchBaseDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config")
}

// guardExeName is the copy of this program the guard runs from.
func guardExeName() string {
	base := filepath.Base(os.Args[0])
	if strings.HasSuffix(strings.ToLower(base), ".exe") {
		return base
	}
	return "freebuff-theme-injector"
}

// ------------------------------------------------------- logon autostart ---

/*
 * The Windows build writes an HKCU Run value. The equivalent that needs no
 * privileges and no display manager is a freedesktop autostart entry: any
 * XDG-compliant desktop session runs it at logon.
 */
func autostartDir() string {
	base := watchBaseDir()
	if base == "" {
		return ""
	}
	return filepath.Join(base, "autostart")
}

func autostartPath() string {
	dir := autostartDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "freebuff-theme-studio.desktop")
}

func setRunEntry(command string) error {
	p := autostartPath()
	if p == "" {
		return errors.New("the home directory could not be determined, so no logon entry was written")
	}
	if err := os.MkdirAll(autostartDir(), 0o755); err != nil {
		return fmt.Errorf("registering the guard to start at logon: %w", err)
	}
	body := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Freebuff Theme Studio guard\n" +
		"Comment=Restores the Theme Studio panel after a Freebuff update\n" +
		"Exec=" + command + "\n" +
		"X-GNOME-Autostart-enabled=true\n" +
		"NoDisplay=true\n" +
		"Terminal=false\n"
	return atomicWriteFile(p, []byte(body), 0o644)
}

func clearRunEntry() {
	if p := autostartPath(); p != "" {
		_ = os.Remove(p) // absent is the normal case during uninstall
	}
}

func runEntryPresent() bool {
	p := autostartPath()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// ------------------------------------------------------------ app storage ---

/*
 * Chromium-based apps on Unix keep their profile under XDG_CONFIG_HOME (or
 * ~/.config) rather than %APPDATA%, and the folder is named after the app.
 */
func cookieRoots() []string {
	var out []string
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		out = append(out, dir)
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".config"))
	}
	return out
}

// bunBinary looks for the Bun runtime the install ships with, which has no
// extension here.
func bunBinary(install string) string {
	for _, p := range []string{
		filepath.Join(install, "resources", "bun", "bun"),
		filepath.Join(install, "resources", "bun", "bun-baseline"),
		filepath.Join(install, "resources", "orchestrator", "bun"),
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("bun"); err == nil {
		return p
	}
	return ""
}
