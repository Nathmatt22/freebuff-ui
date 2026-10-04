# Freebuff Theme Studio — building on Linux and macOS

The injector is a single Go program with no dependencies outside the standard
library. This note covers what had to change to make it build and run on
non-Windows systems, and what to expect there.

## Build

Go 1.21 or newer is the only requirement. No C toolchain: every target is built
with `CGO_ENABLED=0` and is statically linked.

```sh
./build.sh                 # native binary
./build.sh --linux         # Linux amd64  -> dist/FreebuffThemeInjector
./build.sh --linux-arm64   # Linux arm64  -> dist/FreebuffThemeInjector-arm64
./build.sh --darwin        # macOS arm64  -> dist/FreebuffThemeInjector-darwin
./build.sh --windows       # Windows     -> dist/FreebuffThemeInjector.exe
./build.sh --all           # every target
```

If `go` is not on `PATH`, point the script at it:

```sh
GO=/opt/go/bin/go ./build.sh --linux
```

Run the tests with `go test ./...` inside `injector/`.

## How the source is split

`injector/main.go` is platform-neutral and holds all the actual work: locating
an install, injecting the `<script>` tags, the manifest, the backup, the
cookie escape hatch and the guard loop. Everything that differs between
operating systems lives behind a build tag:

| Function | Windows | Linux / macOS |
| --- | --- | --- |
| `replaceFile` | `MoveFileExW` with `REPLACE_EXISTING` | `os.Rename` |
| `freebuffRunning` / `stopFreebuff` | `tasklist` / `taskkill` | `/proc` walk, then `SIGTERM`, then `SIGKILL` |
| `runningExecutable` | PowerShell `Get-Process` | `/proc/<pid>/exe` symlinks |
| `candidateDirs` | `%LOCALAPPDATA%`, `%APPDATA%`, `%ProgramFiles%` | XDG data dirs, `~/.local/share`, `/opt`, `/usr/lib`, snap layout |
| `processAlive` | `os.FindProcess` fails when the pid is gone | `kill(pid, 0)` |
| `processName` | `tasklist` | `/proc/<pid>/comm`, falling back to `cmdline` |
| logon autostart | `HKCU\...\Run` registry value | `~/.config/autostart/*.desktop` |
| `watchBaseDir` | `%LOCALAPPDATA%` | `$XDG_CONFIG_HOME`, else `~/.config` |
| `cookieRoots` | `%APPDATA%`, `%LOCALAPPDATA%` | `$XDG_CONFIG_HOME`, `~/.config` |
| `bunBinary` | `bun.exe` | `bun` |
| `relaunchExe` | `Freebuff.exe` | AppImage, wrapper or bare binary |
| `openFolder` | `explorer.exe` | `xdg-open`, `gio`, `nautilus`, … |
| `enableVT` | enable `ENABLE_VIRTUAL_TERMINAL_PROCESSING` | always true |

Adding a platform means adding one file next to
`injector/platform_windows.go`, with the same function names.

## Freebuff Desktop on Linux is an AppImage

This is the part that matters in practice, and it is not like any other
platform.

Freebuff ships for Linux as a single **AppImage** file, not as an installed
directory. Two consequences:

1. **A running AppImage has no install directory.** It is a squashfs image the
   kernel mounts read-only under `/tmp/.mount_XXXXXX` for as long as it runs.
   The injector therefore finds it through `/proc/<pid>/exe` rather than by
   looking for a folder, and falls back to that mount.
2. **That mount cannot be written to.** The injection would fail with a bare
   permission error halfway through replacing `index.html`.

So the supported way to install the panel on Linux is to extract the AppImage
first, which gives a writable tree:

```sh
chmod +x Freebuff-linux-x86_64.AppImage
./Freebuff-linux-x86_64.AppImage --appimage-extract
./FreebuffThemeInjector --path squashfs-root
./squashfs-root/freebuff          # start Freebuff from the extracted copy
```

Run Freebuff from the extracted folder, not from the AppImage: the panel is
installed into the copy you launch.

If auto-detection finds nothing, the injector says this rather than just
asking for `--path`. If you point `--path` at a read-only mount, it says that
too instead of failing with a permission error.

## What the injector does on Linux

The mechanism is identical to Windows. Freebuff's renderer is served by a Bun
orchestrator bound to `127.0.0.1`, which serves
`resources/orchestrator/ui` straight off disk and computes its CSP from the
document it just read. So writing one extra same-origin `<script>` into
`index.html` is enough, and the panel appears on the next page load. Nothing in
`app.asar` and no binary is modified.

```
./FreebuffThemeInjector                     install the panel
./FreebuffThemeInjector --status            check what is installed
./FreebuffThemeInjector --restart           close and relaunch Freebuff
./FreebuffThemeInjector --theme my.json     bake a theme in as the default
./FreebuffThemeInjector --repair            rewrite the files, keep the theme
./FreebuffThemeInjector --reset-theme       clear stored theme cookies
./FreebuffThemeInjector --uninstall         restore the original index.html
```

If auto-detection does not find the install, pass it explicitly:

```sh
./FreebuffThemeInjector --path /opt/Freebuff
```

### The background guard

An install leaves a small guard behind that re-injects the panel after Freebuff
updates itself, since the updater replaces `resources/orchestrator` wholesale.
It is user-scoped, prints nothing, and is started at logon by a freedesktop
autostart entry in `~/.config/autostart/freebuff-theme-studio.desktop` —
no root, no systemd unit, no display-manager dependency. Its files live in
`~/.config/FreebuffThemeStudio/`.

`--remove-watch` takes all of it away: autostart entry, process and files.

## Caveats

- **The build is verified; the runtime is not, yet.** Every target compiles and
  `go vet` is clean for `linux`, `darwin` and `windows`, and the whole
  install / status / uninstall cycle passes against `sandbox/`. But those tests
  ran on Windows, so the Linux `platform_unix.go` paths — `/proc` scanning, the
  autostart `.desktop` file, `xdg-open`, `SIGTERM` — are exercised only by
  compilation so far. Run it on a real Linux box to confirm.
- **Freebuff Desktop may not ship a Linux build.** This port makes the injector
  cross-platform; it cannot conjure an install to inject into. Detection covers
  the usual locations, but `--path` is the reliable route.
- **The uninstall command shown in the panel adapts.** The injector writes its
  own file name into the script tag's `data-injector-name`, and the theme engine
  reads it back, so the panel shows `./FreebuffThemeInjector --uninstall` on
  Linux and `FreebuffThemeInjector.exe --uninstall` on Windows.