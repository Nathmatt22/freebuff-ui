# Freebuff UI

A colour theme editor for **Freebuff Desktop** for Windows.

Once installed, Freebuff gets a new palette icon in its sidebar. Clicking it opens a
Theme Studio page inside the app, where you can change every colour Freebuff uses,
pick from ready-made themes, and save your own.

You do not need to know how to code to use it.

---

## What you get

- A **palette icon** in Freebuff's left sidebar, next to the other icons.
- A **Theme Studio page** that opens inside Freebuff, the same way its Settings page does.
- **15 ready-made themes**: Nord, Dracula, Tokyo Night, Gruvbox, One Dark, Catppuccin
  (dark and light), Rose Pine, Solarized Light, Cyberpunk, Matrix, Amber CRT, Vaporwave,
  Midnight Blue, High Contrast, and the original Freebuff look.
- **Full colour control**: change any of the 212 colours Freebuff uses, one by one.
- **Layout control**: corner roundness, text size, fonts, and panel sizes.
- **A raw CSS box** for anything the controls do not cover.
- **Import and export** so you can save a theme to a file or share it with someone.

Your theme is remembered after you close and reopen Freebuff.

---

## Requirements

- Windows 10 or 11.
- Freebuff Desktop installed.
- That is all. The installer is a single `.exe` file with nothing else to download.

---

## How to install

1. Download `FreebuffThemeInjector.exe`.
2. Double-click it. A black window appears and prints a short report.
3. When it says **Done**, open Freebuff. If Freebuff is already open, press `Ctrl+R`.

You should now see the palette icon in the sidebar. Click it to open Theme Studio.

The window you see is only open for a few seconds; you can close it by pressing a key.

---

## How to use it

Click the **palette icon** in the sidebar. The Theme Studio page fills the workspace area,
with a list of sections down the left side.

### Presets

Click any theme to apply it instantly to the whole app. The one you are using is
highlighted. Choose **Freebuff Default** to go back to the original colours.

Below the themes are three buttons, Auto, Dark and Light. These tell Freebuff how to
draw things it controls itself, such as scrollbars and dropdown menus. Auto follows
the theme you picked.

### Colors

This is where you fine-tune. Colours are grouped so you can find what you want:

| Group | What it changes |
| --- | --- |
| Surfaces | Backgrounds and panels |
| Text | All text, from headings to faint hints |
| Brand | The accent colour and buttons |
| Borders | Lines and dividers |
| Status | Success, warning, error, and similar colours |
| Syntax | Colours used in code blocks |
| Effects | Shadows and focus outlines |

Each row has a colour swatch, a slider for transparency, a box for typing an exact
colour code, and a small arrow button to reset just that one colour.

Changes appear as you make them. There is no save button.

### Layout

Rounding presets let you go from sharp square corners to very round ones. Text scale
buttons resize all the text at once. Below those you can set the fonts and set exact
sizes for individual elements.

### Advanced

Two things live here:

1. **A raw CSS box.** If you know CSS, you can type any rules you like and they will be
   applied. This can change things the other sections do not reach.
2. **A list of every colour Freebuff has.** There is a search box at the top. Use this
   if you are looking for something specific.

### Export

Give your theme a name, then use the buttons to save it or load one:

- **Import** reads whatever text is in the box as a theme.
- **Download .json** saves your theme as a file you can keep or send to someone.
- **Copy CSS** copies the theme as plain CSS.

### The bottom bar

- **Reset all** puts everything back to the original Freebuff colours.
- The text on the right reminds you where to find this page again.

Press `Esc` to leave the page. Clicking any other sidebar icon also leaves it.
If you ever need to reopen it without the mouse, press `Ctrl+Alt+Shift+F`.

---

## Sharing a theme

A theme file is a small text file like this:

```json
{
  "v": 1,
  "name": "Midnight Blue",
  "preset": "midnight",
  "colors": { "--brand-2": { "hex": "#5b8cff", "a": 1 } },
  "layout": { "--radius-md": "12px" },
  "raw": "",
  "scheme": ""
}
```

To use someone else's theme, open the **Export** section, paste the text into the box,
and press **Import**.

---

## Removing it

Run `FreebuffThemeInjector.exe --uninstall`. This removes the palette icon, deletes the
editor, and puts the original file back exactly as it was. Freebuff returns to normal.

Your saved theme stays in Freebuff's settings, but it stops being applied because the
editor is gone.

---

## If something goes wrong

**The palette icon is not there.**
Press `Ctrl+R` in Freebuff. If that does not help, close Freebuff completely and open it
again. Reloading the page is what makes Freebuff pick up the new file.

**It worked before, then stopped after a Freebuff update.**
Freebuff updates replace the interface file, which removes the icon. Run
`FreebuffThemeInjector.exe` again. It is safe to run as many times as you like; it never
creates duplicates.

**I want to start over.**
Use **Reset all** in the bottom bar of the page.

**Something looks broken and I want it gone.**
Run `FreebuffThemeInjector.exe --uninstall`. This always works, even if the page itself
will not load.

**The installer cannot find Freebuff.**
Point it at the folder yourself with `--path`, for example:

```
FreebuffThemeInjector.exe --path "C:\Users\You\AppData\Local\Programs\@codebufffreebuff-desktop"
```

**Freebuff was open while I installed.**
That is fine. Press `Ctrl+R` in Freebuff. You can also add `--restart` when installing to
have it reopened for you.

---

## All command options

| Option | What it does |
| --- | --- |
| *(no options)* | Install the theme editor |
| `--status` | Check whether it is installed |
| `--uninstall` | Remove it and restore the original file |
| `--theme FILE` | Use a theme file as the starting theme for new sessions |
| `--path DIR` | Target a specific Freebuff folder |
| `--restart` | Reopen Freebuff after installing |
| `--open` | Open the interface folder in File Explorer |
| `--quiet` | Print less |

Examples:

```
FreebuffThemeInjector.exe --restart
FreebuffThemeInjector.exe --theme my-theme.json
FreebuffThemeInjector.exe --status
FreebuffThemeInjector.exe --uninstall
```

---

## For developers

### How it works

Freebuff Desktop is an Electron app. Its window loads a web page from a small local
web server that runs on your machine, and that server reads its interface files straight
from disk every time the page loads.

That gives a clean place to hook in. The installer changes exactly two files inside the
Freebuff folder:

```
resources/orchestrator/ui/index.html                        adds one script tag
resources/orchestrator/ui/assets/freebuff-theme-studio.js   the theme editor
```

It does not modify `app.asar` and it does not patch any binary. A copy of the original
`index.html` is kept next to it as `index.html.freebuff-theme-original.bak`, which is what
`--uninstall` restores.

### How the theming works

Freebuff's entire look is built from 212 CSS custom properties, names like `--bg`,
`--surface`, `--brand-2` and `--syntax-keyword`. The editor sets those as inline styles
on the page's root element, which overrides every stylesheet rule the app ships. Because
many of the app's colours are defined in terms of others, changing a few of them updates
a great deal of the interface automatically.

The editor page itself is rendered in a shadow DOM, so Freebuff's styles cannot affect it,
but it reads its own colours from the app's properties. The result is that the editor is
themed by the theme you apply.

### The sidebar icon

Freebuff renders its sidebar icons as `.shell-nav-button` elements inside
`.shell-navigation-top`. The editor adds one more button there and gives it the same class,
so Freebuff styles it and handles the active highlight. A `MutationObserver`, plus a
repeated check every 1.5 seconds, puts it back if Freebuff ever rebuilds that part of the
interface.

The page is positioned using Freebuff's own measurements, `--tabbar-height` and
`--shell-rail-width`, so it lines up with the workspace area.

### Where your theme is stored

In a cookie. Freebuff chooses a new port every time it starts, and browser storage is tied
to the combination of address and port, so ordinary storage would be lost on every restart.
Cookies are tied to the address only, so they survive.

### Building from source

You need Go 1.21 or newer. There are no other dependencies.

```bash
cd injector
go build -trimpath -ldflags "-s -w" -o ../dist/FreebuffThemeInjector.exe .
```

The editor script in `injector/assets/theme-engine.js` is compiled into the program with
`go:embed`, so the resulting `.exe` is completely self-contained.

### Project layout

```
freebuff-theme-studio/
  injector/
    main.go                    finds Freebuff, installs, checks, uninstalls
    go.mod
    assets/theme-engine.js     the editor page (the real source of truth)
  dist/
    FreebuffThemeInjector.exe  the built program
  sandbox/
    demo.html                  a mock Freebuff shell for previewing the editor
    fake-install/              a fake Freebuff folder for testing the installer safely
```

### Trying it without touching Freebuff

```bash
# from this folder
python -m http.server 8199 --bind 127.0.0.1
# then open http://127.0.0.1:8199/sandbox/demo.html
```

### Testing the installer safely

```bash
./dist/FreebuffThemeInjector.exe --path sandbox/fake-install
./dist/FreebuffThemeInjector.exe --path sandbox/fake-install --status
./dist/FreebuffThemeInjector.exe --path sandbox/fake-install --uninstall
```

---

## Things to know

- A Freebuff update removes the palette icon, because it replaces the interface file.
  Running the installer again brings it back.
- The editor also appears in Freebuff's separate thread windows, which is intended.
- This is an unofficial tool. It is not made by Freebuff.
