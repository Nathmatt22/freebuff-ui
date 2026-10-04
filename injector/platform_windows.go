//go:build windows

// Freebuff Theme Studio - Windows platform layer.
//
// Same function names as platform_unix.go, so main.go stays free of
// OS-specific code.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ---------------------------------------------------------------- console ---

func enableVT() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getStdHandle := kernel32.NewProc("GetStdHandle")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	const stdOutputHandle = ^uintptr(10) // -11
	h, _, _ := getStdHandle.Call(stdOutputHandle)
	if h == 0 || h == uintptr(^uintptr(0)) {
		return false
	}
	var mode uint32
	// GetConsoleMode is exported too; if it fails the handle is not a console.
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	if r, _, _ := getConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode))); r == 0 {
		return false
	}
	const enableVirtualTerminalProcessing = 0x0004
	if r, _, _ := setConsoleMode.Call(h, uintptr(mode|enableVirtualTerminalProcessing)); r == 0 {
		return false
	}
	return true
}

// ------------------------------------------------------------------ files ---

/*
 * MoveFileEx with MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH, which is
 * what os.Rename cannot do here: it refuses to clobber an existing file, and a
 * running image cannot be overwritten at all.
 */
func replaceFile(temp, path string) error {
	src, err := syscall.UTF16PtrFromString(temp)
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	moveFileEx := syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")
	const moveFileReplaceExisting = 0x1
	const moveFileWriteThrough = 0x8
	result, _, callErr := moveFileEx.Call(uintptr(unsafe.Pointer(src)), uintptr(unsafe.Pointer(dst)), moveFileReplaceExisting|moveFileWriteThrough)
	if result == 0 {
		return fmt.Errorf("atomically replacing %s: %w", path, callErr)
	}
	return nil
}

// ------------------------------------------------------------- processes ---

func hideProc(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func detachProc(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: detachedFlag}
}

// An installed Windows tree is always writable, so there is nothing to detect.
func isReadOnlyErr(err error) bool { return false }

func isReadOnlyPath(dir string) bool { return false }

// --------------------------------------------------------------- locating ---

// runningExecutable prefers the live process: it is the install the user runs.
func runningExecutable() string {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`(Get-Process Freebuff -ErrorAction SilentlyContinue | Where-Object { $_.Path } | Select-Object -First 1).Path`)
	hideProc(cmd)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return ""
	}
	return p
}

func candidateDirs() []string {
	var out []string

	if exe := runningExecutable(); exe != "" {
		out = append(out, filepath.Dir(exe))
	}

	local := os.Getenv("LOCALAPPDATA")
	roaming := os.Getenv("APPDATA")
	progFiles := os.Getenv("ProgramFiles")
	progFilesX86 := os.Getenv("ProgramFiles(x86)")

	names := []string{
		"@codebufffreebuff-desktop",
		"freebuff-desktop",
		"Freebuff",
		"Freebuff Desktop",
		"freebuff",
	}
	roots := []string{local, roaming, progFiles, progFilesX86}
	for _, root := range roots {
		if root == "" {
			continue
		}
		for _, n := range names {
			out = append(out, filepath.Join(root, "Programs", n))
			out = append(out, filepath.Join(root, n))
		}
	}

	// Running from the install tree itself.
	if wd, err := os.Getwd(); err == nil {
		out = append(out, wd)
	}
	return out
}

func freebuffRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Freebuff.exe", "/NH")
	hideProc(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "freebuff.exe")
}

func stopFreebuff(quiet bool) bool {
	if !freebuffRunning() {
		return true
	}
	if !quiet {
		info("Asking Freebuff to close…")
	}
	cmd := exec.Command("taskkill", "/IM", "Freebuff.exe")
	hideProc(cmd)
	_ = cmd.Run()
	for i := 0; i < 60; i++ {
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

// relaunchExe: Freebuff.exe lives at the install root.
func relaunchExe(install string) string {
	p := filepath.Join(install, "Freebuff.exe")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func relaunch(install string) {
	exe := relaunchExe(install)
	if exe == "" {
		warn("Could not find %s to relaunch", filepath.Join(install, "Freebuff.exe"))
		return
	}
	info("Launching Freebuff…")
	cmd := exec.Command(exe)
	cmd.Dir = install
	hideProc(cmd)
	if err := cmd.Start(); err != nil {
		warn("Could not launch Freebuff: %v", err)
	}
}

func openFolder(dir string) {
	cmd := exec.Command("explorer.exe", dir)
	hideProc(cmd)
	_ = cmd.Start()
}

// ------------------------------------------------------------------ watch ---

// hideConsoleWindow is what keeps the guard from flashing a console at logon.
func hideConsoleWindow() {
	h, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
	if h != 0 {
		_, _, _ = syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow").Call(h, 0) // SW_HIDE
	}
}

// On Windows os.FindProcess opens the process and fails when the pid is gone.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = p.Release()
	return true
}

func processName(pid int) string {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	hideProc(cmd)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	fields := strings.Split(string(out), ",")
	if len(fields) < 2 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(fields[0]), `"`)
}

func watchBaseDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	return base
}

func guardExeName() string { return "FreebuffThemeInjector.exe" }

// ------------------------------------------------------- logon autostart ---

func runKey() string { return `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` }

func setRunEntry(command string) error {
	cmd := exec.Command("reg", "add", runKey(), "/v", watchRunName, "/t", "REG_SZ", "/d", command, "/f")
	hideProc(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("registering the guard to start at logon: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func clearRunEntry() {
	cmd := exec.Command("reg", "delete", runKey(), "/v", watchRunName, "/f")
	hideProc(cmd)
	_ = cmd.Run() // absent is the normal case during uninstall
}

func runEntryPresent() bool {
	cmd := exec.Command("reg", "query", runKey(), "/v", watchRunName)
	hideProc(cmd)
	out, err := cmd.Output()
	return err == nil && strings.Contains(string(out), watchRunName)
}

// ------------------------------------------------------------ app storage ---

func cookieRoots() []string {
	var out []string
	for _, root := range []string{os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA")} {
		if root != "" {
			out = append(out, root)
		}
	}
	return out
}

func bunBinary(install string) string {
	for _, p := range []string{
		filepath.Join(install, "resources", "bun", "bun.exe"),
		filepath.Join(install, "resources", "bun", "bun-baseline.exe"),
		filepath.Join(install, "resources", "orchestrator", "bun.exe"),
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
