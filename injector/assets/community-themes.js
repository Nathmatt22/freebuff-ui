/*
 * Community themes.
 *
 * These are ordinary Theme Studio themes, and this is the only file a
 * submission has to touch. The easiest way to add one:
 *
 *   1. Build the theme in the editor, then press Copy next to Share code.
 *   2. Paste the code into `shareCode` below, with your name in `author`.
 *   3. Open a pull request against
 *      https://github.com/RichardFlp/freebuff-ui
 *
 * The editor reads `theme` exactly like a .fbtheme file's contents, so a theme
 * pasted here behaves the same as one imported from disk. `shareCode` is
 * accepted too and wins if both are present - it is the shorter thing to paste
 * and it cannot be broken by a stray quote.
 *
 * The file is written next to the engine at install time and loaded before it,
 * so the Community tab works with no network at all.
 */
window.__FREEBUFF_THEME_COMMUNITY__ = {
  format: 1,
  themes: [
    {
      id: 'wintage-golden-default',
      name: 'Wintage — Golden Default',
      author: 'FuneralPixels',
      note: 'Warm gold on charcoal, every corner squared off, and a typewriter face for the code.',
      theme: {
        "v": 1,
        "name": "Wintage — Golden Default",
        "preset": "default",
        "colors": {
          "--bg": { "hex": "#1A1810", "a": 1 },
          "--workspace-surface": { "hex": "#1A1810", "a": 1 },
          "--shell-base": { "hex": "#232018", "a": 1 },
          "--chrome": { "hex": "#232018", "a": 1 },
          "--surface": { "hex": "#332E22", "a": 1 },
          "--surface-2": { "hex": "#3D372A", "a": 1 },
          "--raised": { "hex": "#453D30", "a": 1 },
          "--popover": { "hex": "#453D30", "a": 1 },
          "--input": { "hex": "#332E22", "a": 1 },
          "--bubble": { "hex": "#3D372A", "a": 1 },
          "--control-bg": { "hex": "#332E22", "a": 1 },
          "--control-bg-hover": { "hex": "#3D372A", "a": 1 },
          "--control-bg-pressed": { "hex": "#453D30", "a": 1 },
          "--selected": { "hex": "#3D372A", "a": 1 },
          "--sidebar-canvas": { "hex": "#232018", "a": 1 },
          "--sidebar-row-background": { "hex": "#332E22", "a": 1 },
          "--sidebar-hover": { "hex": "#3D372A", "a": 1 },
          "--sidebar-selected": { "hex": "#3D372A", "a": 1 },
          "--tab-track": { "hex": "#232018", "a": 1 },
          "--tab-active-surface": { "hex": "#332E22", "a": 1 },
          "--terminal-background": { "hex": "#1A1810", "a": 1 },
          "--block-header-bg": { "hex": "#232018", "a": 1 },
          "--catalog-pane": { "hex": "#232018", "a": 1 },
          "--diffs-bg": { "hex": "#14120C", "a": 1 },
          "--fade-surface": { "hex": "#1A1810", "a": 1 },
          "--message-fill": { "hex": "#3D372A", "a": 1 },
          "--message-ink": { "hex": "#D4C89A", "a": 1 },
          "--new-thread-control": { "hex": "#3D372A", "a": 1 },
          "--sb-surface": { "hex": "#232018", "a": 1 },
          "--settings-disabled": { "hex": "#453D30", "a": 1 },
          "--settings-neutral": { "hex": "#3D372A", "a": 1 },
          "--settings-neutral-hover": { "hex": "#453D30", "a": 1 },
          "--settings-switch-track": { "hex": "#453D30", "a": 1 },
          "--settings-tab-track": { "hex": "#332E22", "a": 1 },
          "--text": { "hex": "#D4C89A", "a": 1 },
          "--muted": { "hex": "#9C9371", "a": 1 },
          "--faint": { "hex": "#6E674E", "a": 1 },
          "--placeholder": { "hex": "#6E674E", "a": 1 },
          "--accent": { "hex": "#F0D060", "a": 1 },
          "--accent-dim": { "hex": "#9C9371", "a": 1 },
          "--sidebar-ink": { "hex": "#D4C89A", "a": 1 },
          "--sidebar-muted": { "hex": "#9C9371", "a": 1 },
          "--tab-indicator": { "hex": "#F0D060", "a": 1 },
          "--tab-selected-text": { "hex": "#D4C89A", "a": 1 },
          "--primary-action-text": { "hex": "#1A1810", "a": 1 },
          "--brand-ink": { "hex": "#F0D060", "a": 1 },
          "--sb-headline": { "hex": "#D4C89A", "a": 1 },
          "--sb-ink": { "hex": "#D4C89A", "a": 1 },
          "--settings-disabled-ink": { "hex": "#6E674E", "a": 1 },
          "--settings-neutral-ink": { "hex": "#D4C89A", "a": 1 },
          "--settings-tab-ink": { "hex": "#D4C89A", "a": 1 },
          "--brand-1": { "hex": "#F0D060", "a": 1 },
          "--brand-2": { "hex": "#F0D060", "a": 1 },
          "--brand-3": { "hex": "#75663D", "a": 1 },
          "--brand": { "hex": "#F0D060", "a": 1 },
          "--brand-dim": { "hex": "#5A5040", "a": 1 },
          "--primary-action": { "hex": "#F0D060", "a": 1 },
          "--green": { "hex": "#4A7A20", "a": 1 },
          "--sb-cta-fill": { "hex": "#F0D060", "a": 1 },
          "--sb-cta-text": { "hex": "#1A1810", "a": 1 },
          "--border": { "hex": "#100E08", "a": 1 },
          "--matte-edge": { "hex": "#75663D", "a": 1 },
          "--control-border": { "hex": "#100E08", "a": 1 },
          "--control-border-hover": { "hex": "#F0D060", "a": 1 },
          "--control-highlight": { "hex": "#F0D060", "a": 1 },
          "--control-shadow": { "hex": "#100E08", "a": 1 },
          "--panel-divider": { "hex": "#100E08", "a": 1 },
          "--shell-header-divider": { "hex": "#100E08", "a": 1 },
          "--sidebar-edge": { "hex": "#100E08", "a": 1 },
          "--workspace-edge": { "hex": "#100E08", "a": 1 },
          "--shell-inset": { "hex": "#100E08", "a": 1 },
          "--new-thread-border": { "hex": "#100E08", "a": 1 },
          "--field-focus": { "hex": "#F0D060", "a": 1 },
          "--settings-tab-indicator": { "hex": "#F0D060", "a": 1 },
          "--ok": { "hex": "#4A7A20", "a": 1 },
          "--success-text": { "hex": "#4A7A20", "a": 1 },
          "--warn": { "hex": "#7A7A20", "a": 1 },
          "--warning-text": { "hex": "#7A7A20", "a": 1 },
          "--danger": { "hex": "#7A2020", "a": 1 },
          "--danger-text": { "hex": "#D66464", "a": 1 },
          "--info": { "hex": "#008080", "a": 1 },
          "--premium": { "hex": "#7A7A20", "a": 1 },
          "--conflict": { "hex": "#7A2020", "a": 1 },
          "--merged": { "hex": "#008080", "a": 1 },
          "--settings-danger": { "hex": "#7A2020", "a": 1 },
          "--settings-danger-hover": { "hex": "#D66464", "a": 1 },
          "--syntax-comment": { "hex": "#6E674E", "a": 1 },
          "--syntax-keyword": { "hex": "#F0D060", "a": 1 },
          "--syntax-string": { "hex": "#4A7A20", "a": 1 },
          "--syntax-number": { "hex": "#7A7A20", "a": 1 },
          "--syntax-function": { "hex": "#F0D060", "a": 1 },
          "--syntax-type": { "hex": "#008080", "a": 1 },
          "--syntax-property": { "hex": "#9C9371", "a": 1 },
          "--syntax-punctuation": { "hex": "#D4C89A", "a": 1 },
          "--scrim": { "hex": "#14120C", "a": 0.82 },
          "--shadow": { "hex": "#14120C", "a": 0.88 }
        },
        "layout": {
          "--radius-xs": "0px",
          "--radius-sm": "0px",
          "--radius-md": "0px",
          "--radius-lg": "0px",
          "--radius-xl": "0px",
          "--radius-control": "0px",
          "--radius-chrome-control": "0px",
          "--radius-popup": "0px",
          "--radius-composer": "0px",
          "--radius-dialog": "0px",
          "--radius-block": "0px",
          "--radius-inline": "0px",
          "--font-sans": "Verdana_m1, Verdana, Tahoma, \"MS Sans Serif\", sans-serif",
          "--font-mono": "\"Cascadia Mono\", Consolas, \"Courier New\", monospace"
        },
        "raw": "",
        "scheme": "dark"
      }
    }
  ]
}
