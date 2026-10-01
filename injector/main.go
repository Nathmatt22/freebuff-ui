// Freebuff Theme Studio - injector
//
// Installs a theme customisation panel into a local Freebuff Desktop install.
//
// How it works: Freebuff Desktop's renderer is served by a Bun orchestrator
// bound to 127.0.0.1. That server (`serveSpa`) serves `resources/orchestrator/ui`
// straight off disk on every request, and computes its Content-Security-Policy
// from the document it just read - so an extra same-origin <script> is allowed
// and takes effect on the next page load.
//
// We therefore only ever touch these files inside the install:
//
//	resources/orchestrator/ui/index.html                        (+ injected <script> tags)
//	resources/orchestrator/ui/assets/freebuff-theme-studio.js   (the engine)
//	resources/orchestrator/ui/assets/freebuff-theme-community.js (community theme list)
//
// Nothing is patched in app.asar and no binary is modified.
package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed assets/theme-engine.js
var engineJS []byte

//go:embed assets/community-themes.js
var communityJS []byte

const (
	version       = "1.3.3"
	markerStart   = "<!-- freebuff-theme-studio:start -->"
	markerEnd     = "<!-- freebuff-theme-studio:end -->"
	engineName    = "freebuff-theme-studio.js"
	communityName = "freebuff-theme-community.js"
	defaultName   = "freebuff-theme-default.js"
	// Past this the request header block approaches the 16 KB the app's own
	// server accepts, and every request starts failing.
	cookieDangerBytes = 12000
	backupSuffix      = ".freebuff-theme-original.bak"
	manifestName      = ".freebuff-theme-studio.json"
)

var (
	colBold  = ""
	colDim   = ""
	colGreen = ""
	colRed   = ""
	colCyan  = ""
	colReset = ""
)

// ---------------------------------------------------------------- console ---

func enableVT() bool {
	if runtime.GOOS != "windows" {
		return true
	}
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

func initColors() {
	if enableVT() {
		colBold = "\x1b[1m"
		colDim = "\x1b[2m"
		colGreen = "\x1b[32m"
		colRed = "\x1b[31m"
		colCyan = "\x1b[36m"
		colReset = "\x1b[0m"
	}
}

func banner() {
	fmt.Printf("%s\n", colBold+"Freebuff Theme Studio"+colReset+" "+colDim+"v"+version+colReset)
	fmt.Printf("%s\n", colDim+"Custom colour themes for Freebuff Desktop."+colReset)
	fmt.Printf("%s\n\n", colDim+"Unofficial community extension - not made by Freebuff."+colReset)
}

func ok(msg string, a ...any) {
	fmt.Printf("  %s[ ok ]%s %s\n", colGreen, colReset, fmt.Sprintf(msg, a...))
}
func info(msg string, a ...any) {
	fmt.Printf("  %s[ .. ]%s %s\n", colCyan, colReset, fmt.Sprintf(msg, a...))
}
func warn(msg string, a ...any) {
	fmt.Printf("  %s[ !! ]%s %s\n", colRed, colReset, fmt.Sprintf(msg, a...))
}
func step(msg string, a ...any) { fmt.Printf("\n%s%s%s\n", colBold, fmt.Sprintf(msg, a...), colReset) }

// --------------------------------------------------------------- locating ---

func runningExecutable() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	// Prefer the live process: it is the install the user actually runs.
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`(Get-Process Freebuff -ErrorAction SilentlyContinue | Where-Object { $_.Path } | Select-Object -First 1).Path`)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
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

func isInstallDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "resources", "orchestrator", "ui", "index.html"))
	return err == nil
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

func findInstallDir(override string) (string, error) {
	if override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		if !isInstallDir(abs) {
			return "", fmt.Errorf("%s does not look like a Freebuff install (no resources/orchestrator/ui/index.html)", abs)
		}
		return abs, nil
	}
	seen := map[string]bool{}
	for _, c := range candidateDirs() {
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		if isInstallDir(c) {
			return c, nil
		}
	}
	return "", errors.New("could not find a Freebuff Desktop install; pass --path <install dir>")
}

// ---------------------------------------------------------------- manifest ---

type manifest struct {
	Version       string `json:"version"`
	InstalledAt   string `json:"installedAt"`
	OriginalSHA   string `json:"originalIndexSha256"`
	HadBackup     bool   `json:"hadBackup"`
	BakedThemeSet bool   `json:"bakedThemeSet"`
}

func manifestPath(ui string) string { return filepath.Join(ui, manifestName) }
func indexPath(ui string) string    { return filepath.Join(ui, "index.html") }
func assetsDir(ui string) string    { return filepath.Join(ui, "assets") }
func backupPath(ui string) string   { return indexPath(ui) + backupSuffix }

func readManifest(ui string) *manifest {
	b, err := os.ReadFile(manifestPath(ui))
	if err != nil {
		return nil
	}
	var m manifest
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	return &m
}

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// ------------------------------------------------------------------ inject ---

func scriptBlock(ui string) string {
	nl := "\n"
	raw, err := os.ReadFile(indexPath(ui))
	if err == nil && strings.Contains(string(raw), "\r\n") {
		nl = "\r\n"
	}
	var b strings.Builder
	b.WriteString(markerStart + nl)
	if _, err := os.Stat(filepath.Join(assetsDir(ui), defaultName)); err == nil {
		b.WriteString(`  <script src="./assets/` + defaultName + `"></script>` + nl)
	}
	// The community list is a plain data file and must be loaded first: the
	// engine reads it while it builds the Community tab.
	b.WriteString(`  <script src="./assets/` + communityName + `"></script>` + nl)
	b.WriteString(`  <script src="./assets/` + engineName + `" data-freebuff-theme-studio="` + version + `"></script>` + nl)
	b.WriteString(markerEnd)
	return b.String()
}

// inject is idempotent: re-running replaces the marker block rather than
// stacking a second one.
func inject(ui string, quiet bool) error {
	idx := indexPath(ui)
	original, err := os.ReadFile(idx)
	if err != nil {
		return fmt.Errorf("reading index.html: %w", err)
	}
	html := string(original)

	if !strings.Contains(strings.ToLower(html), "freebuff") && !strings.Contains(html, "startup-recovery") {
		return errors.New("index.html does not look like the Freebuff UI; refusing to modify it")
	}

	// Back up the pristine file exactly once.
	if _, err := os.Stat(backupPath(ui)); os.IsNotExist(err) {
		// Only back up if it is not already injected.
		if !strings.Contains(html, markerStart) {
			if err := os.WriteFile(backupPath(ui), original, 0o644); err != nil {
				return fmt.Errorf("writing backup: %w", err)
			}
			if !quiet {
				ok("Backed up original index.html")
			}
		}
	}

	block := scriptBlock(ui)
	var updated string
	if i := strings.Index(html, markerStart); i != -1 {
		j := strings.Index(html, markerEnd)
		if j == -1 {
			return errors.New("index.html has a broken injection marker; restore the backup and retry")
		}
		updated = html[:i] + block + html[j+len(markerEnd):]
	} else if k := strings.LastIndex(html, "</body>"); k != -1 {
		updated = html[:k] + block + "\n" + html[k:]
	} else {
		updated = html + "\n" + block + "\n"
	}

	if err := os.MkdirAll(assetsDir(ui), 0o755); err != nil {
		return fmt.Errorf("creating assets dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir(ui), engineName), engineJS, 0o644); err != nil {
		return fmt.Errorf("writing engine: %w", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir(ui), communityName), communityJS, 0o644); err != nil {
		return fmt.Errorf("writing community themes: %w", err)
	}
	if err := os.WriteFile(idx, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing index.html: %w", err)
	}

	if err := writeManifest(ui, original); err != nil {
		return err
	}
	return nil
}

func writeManifest(ui string, original []byte) error {
	hadBackup := true
	if _, err := os.Stat(backupPath(ui)); err != nil {
		hadBackup = false
	}
	baked := false
	if _, err := os.Stat(filepath.Join(assetsDir(ui), defaultName)); err == nil {
		baked = true
	}
	m := manifest{
		Version:       version,
		InstalledAt:   time.Now().UTC().Format(time.RFC3339),
		OriginalSHA:   sha256Hex(original),
		HadBackup:     hadBackup,
		BakedThemeSet: baked,
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(manifestPath(ui), b, 0o644)
}

func bakeTheme(ui, themeFile string) error {
	b, err := os.ReadFile(themeFile)
	if err != nil {
		return err
	}
	var probe map[string]any
	if err := json.Unmarshal(b, &probe); err != nil {
		return fmt.Errorf("theme file is not valid JSON: %w", err)
	}
	// A .fbtheme document wraps the theme; bake the theme itself.
	if inner, ok := probe["theme"].(map[string]any); ok {
		_, hasColors := probe["colors"]
		_, hasLayout := probe["layout"]
		if !hasColors && !hasLayout {
			if wrapped, err := json.Marshal(inner); err == nil {
				b = wrapped
			}
		}
	}
	body := "/* baked by FreebuffThemeInjector --theme */\nwindow.__FREEBUFF_THEME_DEFAULT__ = " + string(b) + ";\n"
	return os.WriteFile(filepath.Join(assetsDir(ui), defaultName), []byte(body), 0o644)
}

// --------------------------------------------------------------- uninstall ---

func uninstall(ui string, quiet bool) error {
	idx := indexPath(ui)
	html, err := os.ReadFile(idx)
	if err != nil {
		return err
	}
	s := string(html)
	if i := strings.Index(s, markerStart); i != -1 {
		j := strings.Index(s, markerEnd)
		if j != -1 {
			s = strings.TrimRight(s[:i], "\r\n") + s[j+len(markerEnd):]
			if err := os.WriteFile(idx, []byte(s), 0o644); err != nil {
				return err
			}
		}
	}
	_ = os.Remove(filepath.Join(assetsDir(ui), engineName))
	_ = os.Remove(filepath.Join(assetsDir(ui), communityName))
	_ = os.Remove(filepath.Join(assetsDir(ui), defaultName))
	_ = os.Remove(manifestPath(ui))
	if !quiet {
		ok("Removed injected script tags, engine and manifest")
	}
	if b, err := os.ReadFile(backupPath(ui)); err == nil {
		if err := os.WriteFile(idx, b, 0o644); err != nil {
			return err
		}
		_ = os.Remove(backupPath(ui))
		if !quiet {
			ok("Restored original index.html")
		}
	}
	return nil
}

func status(ui, install string) {
	m := readManifest(ui)
	html, _ := os.ReadFile(indexPath(ui))
	injected := strings.Contains(string(html), markerStart)
	engine := filepath.Join(assetsDir(ui), engineName)
	_, engineErr := os.Stat(engine)

	fmt.Printf("  install   %s\n", ui)
	if injected && engineErr == nil {
		ok("Theme Studio is installed")
	} else {
		warn("Theme Studio is NOT installed")
	}
	if m != nil {
		fmt.Printf("  version   %s\n", m.Version)
		fmt.Printf("  installed %s\n", m.InstalledAt)
	}
	if _, err := os.Stat(backupPath(ui)); err == nil {
		fmt.Printf("  backup    %s\n", backupPath(ui))
	}
	if _, err := os.Stat(filepath.Join(assetsDir(ui), defaultName)); err == nil {
		fmt.Printf("  baked theme file present\n")
	}
	checkThemeCookies(install)
}

// ---------------------------------------------------------- theme cookies ---

/*
 * The theme lives in cookies, because they are the only store that survives a
 * launch: Freebuff serves its UI from a fresh loopback port every time, so
 * localStorage (which is per-origin, and therefore per-port) does not carry
 * over.
 *
 * Cookies have a cost the theme engine has to respect: they are sent with
 * every request, and the Bun server that serves the UI answers HTTP 431 - and
 * the window stays blank and grey - once the request header block passes
 * 16 KB. An early version stored pictures in cookies as well, which could put
 * a quarter of a megabyte into that header, and then no uninstall helped
 * because the cookies were still in the profile.
 *
 * So this is the way out. Chromium keeps its cookie jar in a small SQLite file
 * inside the Freebuff profile; the install ships its own Bun, and Bun reads
 * SQLite natively, so the installer can delete exactly the theme cookies and
 * leave everything else - including Freebuff's own cookies - alone.
 */

const cookieDBScript = `(async () => {
  const { Database } = await import('bun:sqlite')
  const db = new Database(process.env.FREEBUFF_COOKIE_DB)
  const sql = "select count(*) as n, coalesce(sum(length(value)), 0) as b from cookies where name like 'fbts%'"
  const before = db.query(sql).get()
  const out = { cookies: before.n, bytes: before.b, action: process.env.FBTS_COOKIE_ACTION }
  if (process.env.FBTS_COOKIE_ACTION === 'clear') {
    db.run("delete from cookies where name like 'fbts%'")
    out.remaining = db.query(sql).get().n
  }
  console.log(JSON.stringify(out))
})()`

type cookieReport struct {
	Cookies   int    `json:"cookies"`
	Bytes     int    `json:"bytes"`
	Action    string `json:"action"`
	Remaining int    `json:"remaining"`
}

func profileCookieDBs() []string {
	var out []string
	for _, root := range []string{os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA")} {
		if root == "" {
			continue
		}
		for _, name := range []string{"Freebuff", "@codebufffreebuff-desktop", "freebuff-desktop", "Freebuff Desktop"} {
			p := filepath.Join(root, name, "Network", "Cookies")
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			// Only the jar that actually has our cookies in it.
			if bytes.Contains(b, []byte("fbts")) {
				out = append(out, p)
			}
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

// runCookieScript opens a profile cookie jar and either reports or clears the
// theme cookies in it.
func runCookieScript(bun, db, action string) (*cookieReport, error) {
	cmd := exec.Command(bun, "-e", cookieDBScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = append(os.Environ(), "FREEBUFF_COOKIE_DB="+db, "FBTS_COOKIE_ACTION="+action)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	line := strings.TrimSpace(string(out))
	if i := strings.LastIndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[i+1:])
	}
	var rep cookieReport
	if err := json.Unmarshal([]byte(line), &rep); err != nil {
		return nil, fmt.Errorf("could not read the cookie check output: %w", err)
	}
	return &rep, nil
}

// stopFreebuff asks the app to close and waits for it, because Chromium holds
// the cookie jar in memory and would write it back over any changes we make.
func stopFreebuff(quiet bool) bool {
	if !freebuffRunning() {
		return true
	}
	if !quiet {
		info("Asking Freebuff to close\u2026")
	}
	cmd := exec.Command("taskkill", "/IM", "Freebuff.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
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

// clearThemeCookies is the escape hatch for an install whose theme cookies are
// too big for the app to serve its own UI. Returns how many were removed.
func clearThemeCookies(install string, quiet bool) (int, bool) {
	dbs := profileCookieDBs()
	if len(dbs) == 0 {
		return 0, true
	}
	bun := bunBinary(install)
	if bun == "" {
		if !quiet {
			warn("Could not find the Bun runtime that ships with Freebuff, so the cookies were left alone")
			fmt.Printf("       Delete these files by hand: %s\n", strings.Join(dbs, ", "))
		}
		return 0, false
	}
	if !stopFreebuff(quiet) {
		return 0, false
	}
	removed := 0
	for _, db := range dbs {
		rep, err := runCookieScript(bun, db, "clear")
		if err != nil {
			if !quiet {
				warn("Could not open %s: %v", db, err)
			}
			continue
		}
		removed += rep.Cookies
		if !quiet {
			ok("Removed %d theme cookies (%d bytes) from %s", rep.Cookies, rep.Bytes, filepath.Base(filepath.Dir(filepath.Dir(db))))
		}
	}
	return removed, true
}

// checkThemeCookies warns when the stored theme has grown past the size the
// app's own server will accept.
func checkThemeCookies(install string) {
	dbs := profileCookieDBs()
	if len(dbs) == 0 {
		return
	}
	bun := bunBinary(install)
	if bun == "" {
		return
	}
	for _, db := range dbs {
		rep, err := runCookieScript(bun, db, "report")
		if err != nil {
			continue
		}
		if rep.Cookies == 0 {
			continue
		}
		if rep.Bytes > cookieDangerBytes {
			warn("Theme cookies: %d cookies, %d bytes - too big for the app to stay healthy", rep.Cookies, rep.Bytes)
			fmt.Println("       Run with --reset-theme to clear them.")
			continue
		}
		fmt.Printf("  theme     %d cookies, %d bytes\n", rep.Cookies, rep.Bytes)
	}
}

// ------------------------------------------------------------------- misc ---

func freebuffRunning() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Freebuff.exe", "/NH")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "freebuff.exe")
}

func relaunch(ui string) {
	// Freebuff.exe lives at the install root.
	exe := filepath.Join(ui, "Freebuff.exe")
	if _, err := os.Stat(exe); err != nil {
		warn("Could not find %s to relaunch", exe)
		return
	}
	info("Launching Freebuff\u2026")
	cmd := exec.Command(exe)
	cmd.Dir = ui
	_ = cmd.Start()
}

func main() {
	initColors()

	var (
		pathFlag      = flag.String("path", "", "Freebuff install directory (auto-detected by default)")
		uninstallFlag = flag.Bool("uninstall", false, "remove the injected UI and restore the original index.html")
		statusFlag    = flag.Bool("status", false, "show whether Theme Studio is installed")
		themeFlag     = flag.String("theme", "", "path to a .fbtheme or .json theme to bake in as the default")
		resetFlag     = flag.Bool("reset-theme", false, "delete the stored theme cookies (the fix for an app that opens on a blank window)")
		repairFlag    = flag.Bool("repair", false, "reinstall the current files and clear the stored theme cookies")
		restartFlag   = flag.Bool("restart", false, "close Freebuff if it is running and start it again")
		openFlag      = flag.Bool("open", false, "open the UI folder in Explorer")
		quietFlag     = flag.Bool("quiet", false, "less output")
	)
	flag.Usage = func() {
		banner()
		fmt.Printf("Usage: %s [options]\n\n", filepath.Base(os.Args[0]))
		fmt.Println("  Run with no options to install the theme panel into Freebuff Desktop.")
		fmt.Println("  Then open Freebuff and click the palette icon in its sidebar rail.")
		fmt.Println()
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  FreebuffThemeInjector.exe")
		fmt.Println("  FreebuffThemeInjector.exe --restart")
		fmt.Println("  FreebuffThemeInjector.exe --theme my-theme.json")
		fmt.Println("  FreebuffThemeInjector.exe --status")
		fmt.Println("  FreebuffThemeInjector.exe --repair --restart")
		fmt.Println("  FreebuffThemeInjector.exe --uninstall")
		fmt.Println()
		fmt.Println("If Freebuff opens on an empty grey window, run --reset-theme. That is")
		fmt.Println("caused by a theme too big for the app to carry, and clearing it fixes it.")
	}
	flag.Parse()

	quiet := *quietFlag

	if !quiet {
		banner()
	}

	install, err := findInstallDir(*pathFlag)
	if err != nil {
		warn("%v", err)
		os.Exit(1)
	}
	ui := filepath.Join(install, "resources", "orchestrator", "ui")

	switch {
	case *statusFlag:
		status(ui, install)
		return

	case *resetFlag:
		if !quiet {
			step("Clearing the stored theme")
		}
		if _, ok := clearThemeCookies(install, quiet); !ok {
			os.Exit(1)
		}
		fmt.Println()
		ok("Theme cookies cleared. Freebuff should open normally again.")
		if *restartFlag {
			relaunch(install)
		}
		return

	case *uninstallFlag:
		if !quiet {
			step("Removing Theme Studio")
		}
		if err := uninstall(ui, quiet); err != nil {
			warn("%v", err)
			os.Exit(1)
		}
		// The theme cookies are ours, and they are the one thing an uninstall
		// used to leave behind - including the oversized ones that make the app
		// impossible to open. Take them with us.
		clearThemeCookies(install, quiet)
		fmt.Println()
		ok("Freebuff is back to stock. Restart it if it is running.")
		if *restartFlag {
			relaunch(install)
		}
		return
	}

	if !quiet {
		step("Found Freebuff at %s", install)
		if fallback := runningExecutable(); fallback != "" && filepath.Dir(fallback) != install {
			info("A running Freebuff is at %s", filepath.Dir(fallback))
		}
	}

	if *themeFlag != "" {
		if !quiet {
			step("Baking theme %s", *themeFlag)
		}
		if err := os.MkdirAll(assetsDir(ui), 0o755); err != nil {
			warn("%v", err)
			os.Exit(1)
		}
		if err := bakeTheme(ui, *themeFlag); err != nil {
			warn("%v", err)
			os.Exit(1)
		}
		if !quiet {
			ok("Theme will be the default for new sessions")
		}
	}

	if *repairFlag {
		if !quiet {
			step("Repairing the installed files")
		}
	}
	if !quiet {
		step("Installing theme engine")
	}
	if err := inject(ui, quiet); err != nil {
		warn("%v", err)
		os.Exit(1)
	}
	if !quiet {
		ok("Injected into resources/orchestrator/ui/index.html")
		ok("Wrote assets/%s (%d bytes)", engineName, len(engineJS))
		ok("Wrote assets/%s (%d bytes)", communityName, len(communityJS))
	}

	if *repairFlag {
		// The injected files are now whatever this build carries; the stored
		// theme is the other half of a repair, and the only part of it that can
		// stop the app from opening at all.
		clearThemeCookies(install, quiet)
	}

	if *openFlag {
		_ = exec.Command("explorer.exe", ui).Start()
	}

	if quiet {
		fmt.Printf("installed: %s\n", ui)
	} else {
		fmt.Println()
		fmt.Printf("%sDone.%s\n", colBold+colGreen, colReset)
		fmt.Println()
		fmt.Println("  Open Freebuff - a " + colBold + "palette icon" + colReset + " appears in the sidebar rail.")
		fmt.Println("  Click it for presets, per-token colour pickers, layout and raw CSS.")
		fmt.Println()
		fmt.Printf("  %sThe theme is remembered between launches; a background picture lasts for the session.%s\n", colDim, colReset)
		fmt.Printf("  %sBlank grey window? Run this again with --reset-theme.%s\n", colDim, colReset)
		fmt.Printf("  %sCtrl+Alt+Shift+F reopens the page, Esc closes it.%s\n", colDim, colReset)
		fmt.Printf("  %sUnofficial extension: not made by, or endorsed by, Freebuff.%s\n", colDim, colReset)
		fmt.Println()
	}

	if freebuffRunning() {
		if !quiet {
			warn("Freebuff is already running.")
			fmt.Println("       The new panel appears the next time the UI loads - press Ctrl+R in Freebuff,")
			fmt.Println("       or run this again with --restart to relaunch it cleanly.")
		}
		if *restartFlag {
			if !quiet {
				step("Relaunching Freebuff")
			}
			stopFreebuff(quiet)
			relaunch(install)
		}
	} else if *restartFlag {
		if !quiet {
			step("Starting Freebuff")
		}
		relaunch(install)
	}
}
