/*!
 * Freebuff Theme Studio - injected client engine
 *
 * Loaded into the Freebuff Desktop renderer (http://127.0.0.1:<port>/) by
 * FreebuffThemeInjector.exe. Mounts a shadow-DOM control panel that overrides
 * Freebuff's CSS custom properties (the 212 tokens that drive the whole UI).
 *
 * Overrides are applied as INLINE styles on <html>, which outrank every
 * stylesheet rule the app ships (including :root[data-theme=light]).
 *
 * Persistence: cookies are host-scoped (not port-scoped), and Freebuff picks a
 * fresh ephemeral port on every launch, so cookies are the only store that
 * survives a restart. We chunk across several cookies if the theme is large and
 * mirror to localStorage as a same-session fallback.
 */
;(function () {
  'use strict'

  var VERSION = '1.1.0'
  if (window.__FREEBUFF_THEME_STUDIO__) return
  window.__FREEBUFF_THEME_STUDIO__ = VERSION

  /* ------------------------------------------------------------------ *
   * Update channel
   *
   * Freebuff's page CSP has no style-src and no default-src, so a remote
   * stylesheet still loads - but connect-src blocks fetch() to GitHub. A
   * stylesheet is enough though: the file below only *declares* the latest
   * version as a custom property, and getComputedStyle() can read that even
   * when the CSSOM of a cross-origin sheet is off limits. So the check needs
   * neither CORS nor a proxy, and it degrades to silence if it ever fails.
   * ------------------------------------------------------------------ */

  var UPDATE_FEED = 'https://raw.githubusercontent.com/RichardFlp/freebuff-ui/main/update.css'
  var RELEASES_URL = 'https://github.com/RichardFlp/freebuff-ui/releases/tag/v'
  var UPDATE_PROBE = 'data-fbts-update-probe'
  var REMOTE_VERSION_VAR = '--fbts-remote-version'
  var SKIP_COOKIE = 'fbts_update_skip'
  var CHECK_COOKIE = 'fbts_update_check'
  var CHECK_INTERVAL = 6 * 60 * 60 * 1000
  var CHECK_TIMEOUT = 6000

  var COOKIE_PREFIX = 'fbts_theme'
  var COOKIE_COUNT = 'fbts_theme_n'
  var LS_KEY = 'freebuff-theme-studio:v1'
  var RAW_STYLE_ID = 'freebuff-theme-studio-raw'
  var COOKIE_CHUNK = 3200
  var COOKIE_DAYS = 3650

  /* ------------------------------------------------------------------ *
   * Token registry
   * ------------------------------------------------------------------ */

  function t(name, label, group, kind) {
    return { name: name, label: label || name.replace(/^--/, ''), group: group, kind: kind || 'color' }
  }

  var CURATED = [
    // Surfaces
    t('--bg', 'App background', 'Surfaces'),
    t('--workspace-surface', 'Workspace background', 'Surfaces'),
    t('--chrome', 'Window chrome', 'Surfaces'),
    t('--shell-base', 'Shell base', 'Surfaces'),
    t('--surface', 'Panel surface', 'Surfaces'),
    t('--surface-2', 'Panel surface 2', 'Surfaces'),
    t('--raised', 'Raised surface', 'Surfaces'),
    t('--popover', 'Popover / menu', 'Surfaces'),
    t('--input', 'Input field', 'Surfaces'),
    t('--selected', 'Selected row', 'Surfaces'),
    t('--bubble', 'Chat bubble', 'Surfaces'),
    t('--control-bg', 'Control background', 'Surfaces'),
    t('--sidebar-canvas', 'Sidebar background', 'Surfaces'),
    t('--sidebar-row-background', 'Sidebar row', 'Surfaces'),
    t('--sidebar-hover', 'Sidebar row hover', 'Surfaces'),
    t('--sidebar-selected', 'Sidebar row selected', 'Surfaces'),
    t('--tab-track', 'Tab bar track', 'Surfaces'),
    t('--tab-active-surface', 'Active tab', 'Surfaces'),
    t('--terminal-background', 'Terminal background', 'Surfaces'),

    // Text
    t('--text', 'Primary text', 'Text'),
    t('--muted', 'Muted text', 'Text'),
    t('--faint', 'Faint text', 'Text'),
    t('--placeholder', 'Placeholder text', 'Text'),
    t('--accent', 'Accent (text/controls)', 'Text'),
    t('--accent-dim', 'Accent dim', 'Text'),
    t('--sidebar-ink', 'Sidebar text', 'Text'),
    t('--sidebar-muted', 'Sidebar muted text', 'Text'),
    t('--tab-indicator', 'Tab indicator', 'Text'),
    t('--tab-selected-text', 'Active tab text', 'Text'),
    t('--primary-action-text', 'Primary button text', 'Text'),
    t('--brand-ink', 'Brand ink / links', 'Text'),

    // Brand
    t('--brand-1', 'Brand 1 (light)', 'Brand'),
    t('--brand-2', 'Brand 2 (primary)', 'Brand'),
    t('--brand-3', 'Brand 3 (tint)', 'Brand'),
    t('--brand', 'Brand (alias)', 'Brand'),
    t('--brand-dim', 'Brand dim', 'Brand'),
    t('--primary-action', 'Primary action', 'Brand'),
    t('--green', 'Green / success', 'Brand'),

    // Borders & surfaces edges
    t('--border', 'Border', 'Borders'),
    t('--matte-edge', 'Matte edge', 'Borders'),
    t('--control-border', 'Control border', 'Borders'),
    t('--control-border-hover', 'Control border hover', 'Borders'),
    t('--panel-divider', 'Panel divider', 'Borders'),
    t('--shell-header-divider', 'Shell header divider', 'Borders'),
    t('--sidebar-edge', 'Sidebar edge', 'Borders'),
    t('--workspace-edge', 'Workspace edge', 'Borders'),
    t('--shell-inset', 'Shell inset', 'Borders'),
    t('--scrim', 'Scrim / overlay', 'Borders'),

    // Status
    t('--ok', 'OK', 'Status'),
    t('--success-text', 'Success text', 'Status'),
    t('--warn', 'Warning', 'Status'),
    t('--warning-text', 'Warning text', 'Status'),
    t('--danger', 'Danger', 'Status'),
    t('--danger-text', 'Danger text', 'Status'),
    t('--info', 'Info', 'Status'),
    t('--premium', 'Premium / pro', 'Status'),
    t('--conflict', 'Conflict', 'Status'),
    t('--merged', 'Merged', 'Status'),

    // Syntax
    t('--syntax-comment', 'Comment', 'Syntax'),
    t('--syntax-keyword', 'Keyword', 'Syntax'),
    t('--syntax-string', 'String', 'Syntax'),
    t('--syntax-number', 'Number', 'Syntax'),
    t('--syntax-function', 'Function', 'Syntax'),
    t('--syntax-type', 'Type', 'Syntax'),
    t('--syntax-property', 'Property', 'Syntax'),
    t('--syntax-punctuation', 'Punctuation', 'Syntax'),

    // Effects
    t('--shadow', 'Shadow color', 'Effects'),
    t('--shadow-sm', 'Small shadow', 'Effects', 'text'),
    t('--shadow-md', 'Medium shadow', 'Effects', 'text'),
    t('--shadow-lg', 'Large shadow', 'Effects', 'text'),
    t('--workspace-shadow', 'Workspace shadow', 'Effects', 'text'),
    t('--focus-ring', 'Focus ring', 'Effects', 'text'),
    t('--logo-filter', 'Logo filter', 'Effects', 'text'),

    // Layout
    t('--radius-xs', 'Radius XS', 'Layout', 'size'),
    t('--radius-sm', 'Radius SM', 'Layout', 'size'),
    t('--radius-md', 'Radius MD', 'Layout', 'size'),
    t('--radius-lg', 'Radius LG', 'Layout', 'size'),
    t('--radius-xl', 'Radius XL', 'Layout', 'size'),
    t('--radius-control', 'Radius control', 'Layout', 'size'),
    t('--radius-chrome-control', 'Radius chrome control', 'Layout', 'size'),
    t('--radius-popup', 'Radius popup', 'Layout', 'size'),
    t('--radius-composer', 'Radius composer', 'Layout', 'size'),
    t('--radius-dialog', 'Radius dialog', 'Layout', 'size'),
    t('--radius-block', 'Radius block', 'Layout', 'size'),
    t('--radius-inline', 'Radius inline', 'Layout', 'size'),

    // Typography
    t('--font-sans', 'UI font', 'Typography', 'text'),
    t('--font-mono', 'Mono font', 'Typography', 'text'),
    t('--font-size-caption', 'Size caption', 'Typography', 'size'),
    t('--font-size-label', 'Size label', 'Typography', 'size'),
    t('--font-size-ui', 'Size UI', 'Typography', 'size'),
    t('--font-size-body', 'Size body', 'Typography', 'size'),
    t('--font-size-message', 'Size message', 'Typography', 'size'),
    t('--font-size-title', 'Size title', 'Typography', 'size'),
    t('--font-size-heading', 'Size heading', 'Typography', 'size'),
    t('--font-size-display', 'Size display', 'Typography', 'size'),

    // Sizing
    t('--chat-content-max-width', 'Chat max width', 'Sizing', 'size'),
    t('--explorer-width', 'Explorer width', 'Sizing', 'size'),
    t('--catalog-width', 'Catalog width', 'Sizing', 'size'),
    t('--tabbar-height', 'Tab bar height', 'Sizing', 'size'),
    t('--control-height-md', 'Control height', 'Sizing', 'size'),
    t('--sidebar-row-gap', 'Sidebar row gap', 'Sizing', 'size'),
  ]

  var GROUP_ORDER = [
    'Surfaces',
    'Text',
    'Brand',
    'Borders',
    'Status',
    'Syntax',
    'Effects',
    'Layout',
    'Typography',
    'Sizing',
  ]

  // Every custom property Freebuff ships (scraped from its stylesheet).
  // Exposed in the Advanced tab so nothing is off-limits.
  var ALL_TOKENS = [
    '--accent', '--accent-dim', '--app-corner-inset', '--bg', '--block-header-bg', '--block-shadow',
    '--border', '--brand', '--brand-1', '--brand-2', '--brand-3', '--brand-dim', '--brand-ink',
    '--browser-menu-top', '--bubble', '--catalog-pane', '--catalog-width', '--chat-content-max-width',
    '--chat-gutter', '--chrome', '--clip-radius', '--conflict', '--control-bg', '--control-bg-hover',
    '--control-bg-pressed', '--control-border', '--control-border-hover', '--control-height-lg',
    '--control-height-md', '--control-height-sm', '--control-highlight', '--control-shadow', '--danger',
    '--danger-text', '--diffs-bg', '--diffs-font-family', '--diffs-font-size', '--diffs-gap-block',
    '--diffs-gap-inline', '--diffs-line-height', '--explorer-reserve', '--explorer-visible-width',
    '--explorer-width', '--fade-bottom', '--fade-left', '--fade-right', '--fade-surface', '--fade-top',
    '--faint', '--field-focus', '--focus-halo', '--focus-ring', '--font-mono', '--font-sans',
    '--font-size-body', '--font-size-caption', '--font-size-display', '--font-size-heading',
    '--font-size-label', '--font-size-message', '--font-size-sb-action', '--font-size-sb-headline',
    '--font-size-sb-initial', '--font-size-sb-label', '--font-size-sb-subhead',
    '--font-size-showcase-close', '--font-size-showcase-close-xs', '--font-size-showcase-cta',
    '--font-size-showcase-cta-sm', '--font-size-showcase-label', '--font-size-showcase-sub',
    '--font-size-showcase-sub-md', '--font-size-showcase-sub-sm', '--font-size-showcase-title',
    '--font-size-showcase-title-md', '--font-size-showcase-title-sm', '--font-size-showcase-title-xs',
    '--font-size-title', '--font-size-ui', '--font-weight-bold', '--font-weight-light',
    '--font-weight-medium', '--font-weight-regular', '--font-weight-semibold', '--green', '--hold-tint',
    '--info', '--input', '--logo-filter', '--matte-edge', '--merged', '--message-fill', '--message-ink',
    '--muted', '--native-controls-width', '--new-thread-border', '--new-thread-control', '--ok',
    '--panel-divider', '--placeholder', '--popover', '--premium', '--primary-action',
    '--primary-action-text', '--project-motion-duration', '--project-motion-ease', '--radius',
    '--radius-block', '--radius-chrome-control', '--radius-composer', '--radius-control', '--radius-dialog',
    '--radius-inline', '--radius-lg', '--radius-md', '--radius-popup', '--radius-round', '--radius-sm',
    '--radius-xl', '--radius-xs', '--rail-reserve', '--raised', '--resume-tint', '--sb-cta-fill',
    '--sb-cta-text', '--sb-headline', '--sb-ink', '--sb-surface', '--scrim', '--selected',
    '--settings-card-radius', '--settings-control-radius', '--settings-corner', '--settings-danger',
    '--settings-danger-hover', '--settings-disabled', '--settings-disabled-ink', '--settings-neutral',
    '--settings-neutral-hover', '--settings-neutral-ink', '--settings-popup-radius',
    '--settings-sidebar-width', '--settings-switch-track', '--settings-tab-indicator', '--settings-tab-ink',
    '--settings-tab-track', '--shadow', '--shadow-lg', '--shadow-md', '--shadow-sm', '--shell-base',
    '--shell-header-divider', '--shell-inset', '--shell-rail-width', '--sidebar-canvas', '--sidebar-edge',
    '--sidebar-hover', '--sidebar-ink', '--sidebar-muted', '--sidebar-row-background',
    '--sidebar-row-font-size', '--sidebar-row-font-weight', '--sidebar-row-gap', '--sidebar-row-radius',
    '--sidebar-selected', '--space-0', '--space-1', '--space-2', '--space-3', '--space-4', '--space-5',
    '--space-6', '--space-7', '--space-8', '--splash-fade', '--success-text', '--surface', '--surface-2',
    '--syntax-comment', '--syntax-function', '--syntax-keyword', '--syntax-number', '--syntax-property',
    '--syntax-punctuation', '--syntax-string', '--syntax-type', '--tab-active-surface', '--tab-gap',
    '--tab-height', '--tab-indicator', '--tab-new-space', '--tab-selected-text', '--tab-track',
    '--tabbar-height', '--tabbar-inset', '--tail-clearance', '--terminal-background', '--text',
    '--thinking-reveal', '--tint', '--title-fade', '--transcript-fade-bottom', '--transcript-fade-top',
    '--transcript-pad-top', '--update-line-height', '--user-line-height', '--warn', '--warning-text',
    '--workspace-corner', '--workspace-edge', '--workspace-shadow', '--workspace-surface',
  ]

  /* ------------------------------------------------------------------ *
   * Presets
   * ------------------------------------------------------------------ */

  // Compact palette -> full token map. Keeps each preset readable.
  function palette(p) {
    var out = {
      '--bg': p.bg,
      '--workspace-surface': p.bg,
      '--shell-base': p.shell || p.bg,
      '--chrome': p.shell || p.bg,
      '--surface': p.surface,
      '--surface-2': p.surface2,
      '--raised': p.raised || p.surface2,
      '--popover': p.raised || p.surface2,
      '--input': p.surface2,
      '--bubble': p.surface2,
      '--control-bg': p.surface2,
      '--border': p.border,
      '--matte-edge': p.edge,
      '--control-border': p.edge,
      '--panel-divider': p.border,
      '--shell-header-divider': p.border,
      '--sidebar-edge': p.border,
      '--workspace-edge': p.border,
      '--text': p.text,
      '--muted': p.muted,
      '--faint': p.faint,
      '--placeholder': p.faint,
      '--accent': p.accent,
      '--accent-dim': p.accentDim,
      '--sidebar-ink': p.text,
      '--sidebar-muted': p.muted,
      '--brand-1': p.brand1 || p.brand,
      '--brand-2': p.brand2 || p.brand,
      '--brand-3': p.brand3 || p.brand,
      '--brand': p.brand,
      '--brand-dim': p.brandDim,
      '--brand-ink': p.brandInk || p.brand,
      '--primary-action': p.primary || p.brand,
      '--primary-action-text': p.primaryText || p.bg,
      '--green': p.ok,
      '--ok': p.ok,
      '--success-text': p.ok,
      '--warn': p.warn,
      '--warning-text': p.warnText || p.warn,
      '--danger': p.danger,
      '--danger-text': p.dangerText || p.danger,
      '--info': p.info,
      '--premium': p.premium || p.warn,
      '--conflict': p.warn,
      '--merged': p.merged || p.info,
      '--selected': p.selected || p.raised || p.surface2,
      '--tab-track': p.shell || p.bg,
      '--tab-active-surface': p.surface,
      '--tab-indicator': p.text,
      '--tab-selected-text': p.bg,
      '--sidebar-canvas': p.shell || p.bg,
      '--sidebar-row-background': p.surface,
      '--sidebar-hover': p.surface2,
      '--sidebar-selected': p.raised || p.surface2,
      '--terminal-background': p.terminal || p.bg,
      '--scrim': p.scrim,
      '--logo-filter': p.logoFilter || 'none',
      '--shadow': p.scrim,
      '--syntax-comment': p.syn.comment,
      '--syntax-keyword': p.syn.keyword,
      '--syntax-string': p.syn.string,
      '--syntax-number': p.syn.number,
      '--syntax-function': p.syn.fn,
      '--syntax-type': p.syn.type,
      '--syntax-property': p.syn.prop,
      '--syntax-punctuation': p.syn.punct,
    }
    for (var k in out) if (out[k] === undefined || out[k] === null) delete out[k]
    return out
  }

  var PRESETS = [
    { id: 'default', label: 'Freebuff Default', scheme: 'dark', colors: null },

    {
      id: 'midnight', label: 'Midnight Blue', scheme: 'dark',
      colors: palette({
        bg: '#0b1220', shell: '#0e1626', surface: '#131c2e', surface2: '#1a2438', raised: '#212d45',
        border: '#243149', edge: 'rgba(120,170,255,0.10)', text: '#e2e8f4', muted: '#94a3bd',
        faint: '#6b7b96', accent: '#8ab4ff', accentDim: '#5c7ba8', brand: '#5b8cff', brandDim: '#4a75d8',
        primaryText: '#08101f', ok: '#4ade80', warn: '#fbbf24', danger: '#f87171', info: '#60a5fa',
        merged: '#c084fc', scrim: 'rgba(0,0,0,0.72)',
        syn: { comment: '#5a6b85', keyword: '#8ab4ff', string: '#7dd3a0', number: '#f0a868', fn: '#7cc7ff', type: '#c9a6f2', prop: '#8fd6cf', punct: '#94a3bd' },
      }),
    },
    {
      id: 'nord', label: 'Nord', scheme: 'dark',
      colors: palette({
        bg: '#2e3440', shell: '#2b303b', surface: '#3b4252', surface2: '#434c5e', raised: '#4c566a',
        border: '#4c566a', edge: 'rgba(216,222,233,0.10)', text: '#eceff4', muted: '#d8dee9',
        faint: '#aebacf', accent: '#88c0d0', accentDim: '#81a1c1', brand: '#88c0d0', brandDim: '#8fbcbb',
        primaryText: '#2e3440', ok: '#a3be8c', warn: '#ebcb8b', danger: '#bf616a', info: '#81a1c1',
        merged: '#b48ead', scrim: 'rgba(0,0,0,0.7)',
        syn: { comment: '#616e88', keyword: '#81a1c1', string: '#a3be8c', number: '#b48ead', fn: '#88c0d0', type: '#8fbcbb', prop: '#d8dee9', punct: '#eceff4' },
      }),
    },
    {
      id: 'dracula', label: 'Dracula', scheme: 'dark',
      colors: palette({
        bg: '#21222c', shell: '#1e1f29', surface: '#282a36', surface2: '#343746', raised: '#44475a',
        border: '#44475a', edge: 'rgba(255,255,255,0.10)', text: '#f8f8f2', muted: '#bfc2cf',
        faint: '#8b8fa8', accent: '#bd93f9', accentDim: '#9d78d6', brand: '#bd93f9', brandDim: '#a679e0',
        primaryText: '#1e1f29', ok: '#50fa7b', warn: '#f1fa8c', danger: '#ff5555', info: '#8be9fd',
        merged: '#ff79c6', scrim: 'rgba(0,0,0,0.72)',
        syn: { comment: '#6272a4', keyword: '#ff79c6', string: '#f1fa8c', number: '#bd93f9', fn: '#50fa7b', type: '#8be9fd', prop: '#f8f8f2', punct: '#ff79c6' },
      }),
    },
    {
      id: 'tokyo-night', label: 'Tokyo Night', scheme: 'dark',
      colors: palette({
        bg: '#1a1b26', shell: '#16161e', surface: '#1f2335', surface2: '#292e42', raised: '#343b58',
        border: '#2f334d', edge: 'rgba(122,162,247,0.10)', text: '#c0caf5', muted: '#9aa5ce',
        faint: '#565f89', accent: '#7aa2f7', accentDim: '#3d59a1', brand: '#7aa2f7', brandDim: '#bb9af7',
        primaryText: '#16161e', ok: '#9ece6a', warn: '#e0af68', danger: '#f7768e', info: '#7dcfff',
        merged: '#bb9af7', scrim: 'rgba(0,0,0,0.75)',
        syn: { comment: '#565f89', keyword: '#bb9af7', string: '#9ece6a', number: '#ff9e64', fn: '#7aa2f7', type: '#2ac3de', prop: '#73daca', punct: '#89ddff' },
      }),
    },
    {
      id: 'gruvbox', label: 'Gruvbox Dark', scheme: 'dark',
      colors: palette({
        bg: '#1d2021', shell: '#181a1b', surface: '#282828', surface2: '#32302f', raised: '#3c3836',
        border: '#504945', edge: 'rgba(235,219,178,0.10)', text: '#ebdbb2', muted: '#bdae93',
        faint: '#928374', accent: '#fabd2f', accentDim: '#d79921', brand: '#fabd2f', brandDim: '#d65d0e',
        primaryText: '#1d2021', ok: '#b8bb26', warn: '#fabd2f', danger: '#fb4934', info: '#83a598',
        merged: '#d3869b', scrim: 'rgba(0,0,0,0.72)',
        syn: { comment: '#928374', keyword: '#fb4934', string: '#b8bb26', number: '#d3869b', fn: '#8ec07c', type: '#fabd2f', prop: '#83a598', punct: '#ebdbb2' },
      }),
    },
    {
      id: 'one-dark', label: 'One Dark', scheme: 'dark',
      colors: palette({
        bg: '#1e2227', shell: '#1a1d21', surface: '#21252b', surface2: '#282c34', raised: '#333842',
        border: '#3a4049', edge: 'rgba(171,178,191,0.10)', text: '#abb2bf', muted: '#8b929e',
        faint: '#6b727d', accent: '#61afef', accentDim: '#528bff', brand: '#61afef', brandDim: '#4d78cc',
        primaryText: '#1e2227', ok: '#98c379', warn: '#e5c07b', danger: '#e06c75', info: '#56b6c2',
        merged: '#c678dd', scrim: 'rgba(0,0,0,0.72)',
        syn: { comment: '#5c6370', keyword: '#c678dd', string: '#98c379', number: '#d19a66', fn: '#61afef', type: '#e5c07b', prop: '#e06c75', punct: '#abb2bf' },
      }),
    },
    {
      id: 'catppuccin-mocha', label: 'Catppuccin Mocha', scheme: 'dark',
      colors: palette({
        bg: '#11111b', shell: '#181825', surface: '#1e1e2e', surface2: '#313244', raised: '#45475a',
        border: '#45475a', edge: 'rgba(205,214,244,0.10)', text: '#cdd6f4', muted: '#a6adc8',
        faint: '#7f849c', accent: '#89b4fa', accentDim: '#74a0e0', brand: '#cba6f7', brandDim: '#89b4fa',
        primaryText: '#11111b', ok: '#a6e3a1', warn: '#f9e2af', danger: '#f38ba8', info: '#89dceb',
        merged: '#f5c2e7', scrim: 'rgba(0,0,0,0.75)',
        syn: { comment: '#6c7086', keyword: '#cba6f7', string: '#a6e3a1', number: '#fab387', fn: '#89b4fa', type: '#f9e2af', prop: '#89dceb', punct: '#cdd6f4' },
      }),
    },
    {
      id: 'rose-pine', label: 'Rose Pine', scheme: 'dark',
      colors: palette({
        bg: '#191724', shell: '#1f1d2e', surface: '#1f1d2e', surface2: '#26233a', raised: '#403d52',
        border: '#403d52', edge: 'rgba(224,222,244,0.10)', text: '#e0def4', muted: '#a9a5c0',
        faint: '#6e6a86', accent: '#c4a7e7', accentDim: '#9ccfd8', brand: '#ebbcba', brandDim: '#c4a7e7',
        primaryText: '#191724', ok: '#31748f', warn: '#f6c177', danger: '#eb6f92', info: '#9ccfd8',
        merged: '#c4a7e7', scrim: 'rgba(0,0,0,0.72)',
        syn: { comment: '#6e6a86', keyword: '#c4a7e7', string: '#f6c177', number: '#ebbcba', fn: '#9ccfd8', type: '#31748f', prop: '#eb6f92', punct: '#e0def4' },
      }),
    },
    {
      id: 'cyberpunk', label: 'Cyberpunk Neon', scheme: 'dark',
      colors: palette({
        bg: '#0a0a12', shell: '#08080f', surface: '#12121f', surface2: '#1b1b2e', raised: '#26264a',
        border: '#3a1f5c', edge: 'rgba(0,255,240,0.14)', text: '#e6f7ff', muted: '#9fb8d0',
        faint: '#6b7f96', accent: '#00fff0', accentDim: '#00b3a8', brand: '#ff2e97', brandDim: '#00fff0',
        primaryText: '#08080f', ok: '#00ff9d', warn: '#ffd400', danger: '#ff3860', info: '#00d9ff',
        merged: '#c77dff', scrim: 'rgba(0,0,0,0.8)',
        syn: { comment: '#5b6b7f', keyword: '#ff2e97', string: '#00ff9d', number: '#ffd400', fn: '#00fff0', type: '#c77dff', prop: '#00d9ff', punct: '#9fb8d0' },
      }),
    },
    {
      id: 'matrix', label: 'Matrix Terminal', scheme: 'dark',
      colors: palette({
        bg: '#000a00', shell: '#000600', surface: '#04140a', surface2: '#08210f', raised: '#0d3016',
        border: '#1a4d24', edge: 'rgba(0,255,65,0.16)', text: '#b9ffb9', muted: '#5fbf6a',
        faint: '#3c8c46', accent: '#00ff41', accentDim: '#00b32e', brand: '#00ff41', brandDim: '#00c734',
        primaryText: '#000a00', ok: '#00ff41', warn: '#b6ff00', danger: '#ff5f5f', info: '#00ffa3',
        merged: '#7dff8a', scrim: 'rgba(0,20,0,0.8)',
        syn: { comment: '#2f7a38', keyword: '#00ff41', string: '#a6ff4d', number: '#7dff8a', fn: '#00ff9c', type: '#b6ff00', prop: '#00d97e', punct: '#5fbf6a' },
      }),
    },
    {
      id: 'amber-crt', label: 'Amber CRT', scheme: 'dark',
      colors: palette({
        bg: '#120b03', shell: '#0d0802', surface: '#1c1206', surface2: '#26190a', raised: '#332211',
        border: '#43301a', edge: 'rgba(255,176,0,0.16)', text: '#ffd9a0', muted: '#c79449',
        faint: '#8c6a33', accent: '#ffb000', accentDim: '#c98a00', brand: '#ffb000', brandDim: '#e09a00',
        primaryText: '#120b03', ok: '#ffd166', warn: '#ff8c00', danger: '#ff6b35', info: '#ffcf70',
        merged: '#ffa94d', scrim: 'rgba(20,10,0,0.8)',
        syn: { comment: '#8c6a33', keyword: '#ffb000', string: '#ffd166', number: '#ff8c00', fn: '#ffc04d', type: '#ffdf9e', prop: '#e0a33a', punct: '#c79449' },
      }),
    },
    {
      id: 'solarized-light', label: 'Solarized Light', scheme: 'light',
      colors: palette({
        bg: '#fdf6e3', shell: '#eee8d5', surface: '#fdf6e3', surface2: '#eee8d5', raised: '#e3ddc9',
        border: '#ded8c4', edge: 'rgba(88,110,117,0.14)', text: '#073642', muted: '#586e75',
        faint: '#93a1a1', accent: '#268bd2', accentDim: '#586e75', brand: '#268bd2', brandDim: '#2aa198',
        primaryText: '#fdf6e3', ok: '#859900', warn: '#b58900', danger: '#dc322f', info: '#2aa198',
        merged: '#d33682', scrim: 'rgba(0,43,54,0.35)', logoFilter: 'brightness(0) invert(0.15)',
        syn: { comment: '#93a1a1', keyword: '#859900', string: '#2aa198', number: '#d33682', fn: '#268bd2', type: '#b58900', prop: '#268bd2', punct: '#586e75' },
      }),
    },
    {
      id: 'catppuccin-latte', label: 'Catppuccin Latte', scheme: 'light',
      colors: palette({
        bg: '#e6e9ef', shell: '#dce0e8', surface: '#eff1f5', surface2: '#e6e9ef', raised: '#dce0e8',
        border: '#d3d7e0', edge: 'rgba(76,79,105,0.12)', text: '#4c4f69', muted: '#6c6f85',
        faint: '#8c8fa1', accent: '#1e66f5', accentDim: '#7287fd', brand: '#8839ef', brandDim: '#1e66f5',
        primaryText: '#eff1f5', ok: '#40a02b', warn: '#df8e1d', danger: '#d20f39', info: '#04a5e5',
        merged: '#ea76cb', scrim: 'rgba(76,79,105,0.3)', logoFilter: 'brightness(0) invert(0.2)',
        syn: { comment: '#9ca0b0', keyword: '#8839ef', string: '#40a02b', number: '#fe640b', fn: '#1e66f5', type: '#df8e1d', prop: '#04a5e5', punct: '#6c6f85' },
      }),
    },
    {
      id: 'high-contrast', label: 'High Contrast', scheme: 'dark',
      colors: palette({
        bg: '#000000', shell: '#000000', surface: '#0a0a0a', surface2: '#141414', raised: '#1f1f1f',
        border: '#6b6b6b', edge: 'rgba(255,255,255,0.5)', text: '#ffffff', muted: '#e0e0e0',
        faint: '#bdbdbd', accent: '#ffffff', accentDim: '#c9c9c9', brand: '#ffe600', brandDim: '#ffd000',
        primaryText: '#000000', ok: '#3cff7a', warn: '#ffd000', danger: '#ff4d4d', info: '#6ec8ff',
        merged: '#e58cff', scrim: 'rgba(0,0,0,0.9)', logoFilter: 'brightness(0) invert(1)',
        syn: { comment: '#9e9e9e', keyword: '#ff9cf0', string: '#7dff9e', number: '#ffcc66', fn: '#8ecbff', type: '#ffe066', prop: '#7fe8dd', punct: '#ffffff' },
      }),
    },
    {
      id: 'vaporwave', label: 'Vaporwave', scheme: 'dark',
      colors: palette({
        bg: '#1a1033', shell: '#140b28', surface: '#241547', surface2: '#2f1b5c', raised: '#3d2475',
        border: '#4b2c8f', edge: 'rgba(255,113,206,0.16)', text: '#f5e6ff', muted: '#c0a3e0',
        faint: '#8f74b8', accent: '#ff71ce', accentDim: '#b967ff', brand: '#01cdfe', brandDim: '#ff71ce',
        primaryText: '#1a1033', ok: '#05ffa1', warn: '#fffb96', danger: '#ff3d6e', info: '#01cdfe',
        merged: '#b967ff', scrim: 'rgba(10,4,24,0.78)',
        syn: { comment: '#7a6ba8', keyword: '#ff71ce', string: '#05ffa1', number: '#fffb96', fn: '#01cdfe', type: '#b967ff', prop: '#84ffe1', punct: '#c0a3e0' },
      }),
    },
  ]

  var PRESET_BY_ID = {}
  PRESETS.forEach(function (p) {
    PRESET_BY_ID[p.id] = p
  })

  /* ------------------------------------------------------------------ *
   * Color helpers
   * ------------------------------------------------------------------ */

  function clamp255(n) {
    return Math.max(0, Math.min(255, Math.round(n)))
  }

  function parseColor(input) {
    if (!input) return null
    var s = String(input).trim().toLowerCase()
    var m = /^#([0-9a-f]{3,8})$/.exec(s)
    if (m) {
      var h = m[1]
      if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2]
      if (h.length === 4) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2] + h[3] + h[3]
      if (h.length !== 6 && h.length !== 8) return null
      return {
        hex: '#' + h.slice(0, 6),
        a: h.length === 8 ? parseInt(h.slice(6, 8), 16) / 255 : 1,
      }
    }
    m = /^rgba?\(([^)]+)\)$/.exec(s)
    if (m) {
      var parts = m[1].split(/[\s,/]+/).filter(Boolean)
      if (parts.length < 3) return null
      var r = parseFloat(parts[0])
      var g = parseFloat(parts[1])
      var b = parseFloat(parts[2])
      var a = parts.length > 3 ? parseFloat(parts[3]) : 1
      if (isNaN(r) || isNaN(g) || isNaN(b)) return null
      return { hex: rgbToHex(r, g, b), a: isNaN(a) ? 1 : a }
    }
    return null
  }

  function rgbToHex(r, g, b) {
    return (
      '#' +
      [r, g, b]
        .map(function (v) {
          return clamp255(v).toString(16).padStart(2, '0')
        })
        .join('')
    )
  }

  function toCss(color) {
    if (!color) return ''
    if (typeof color === 'string') return color
    if (!color.a || color.a >= 1) return color.hex
    var r = parseInt(color.hex.slice(1, 3), 16)
    var g = parseInt(color.hex.slice(3, 5), 16)
    var b = parseInt(color.hex.slice(5, 7), 16)
    return 'rgb(' + r + ' ' + g + ' ' + b + ' / ' + Math.round(color.a * 100) + '%)'
  }

  /** Resolve a token to a concrete rgb()/rgba() by letting the browser compute it. */
  var probeEl = null
  function resolveTokenColor(token) {
    if (!probeEl) {
      probeEl = document.createElement('span')
      probeEl.style.cssText = 'position:absolute;left:-9999px;top:-9999px;width:0;height:0'
      ;(document.body || document.documentElement).appendChild(probeEl)
    }
    probeEl.style.color = ''
    probeEl.style.color = 'var(' + token + ')'
    var computed = getComputedStyle(probeEl).color || ''
    var parsed = parseColor(computed)
    if (parsed) return parsed
    return parseColor(getComputedStyle(document.documentElement).getPropertyValue(token))
  }

  /* ------------------------------------------------------------------ *
   * State + persistence
   * ------------------------------------------------------------------ */

  var DEFAULT_STATE = {
    v: 1,
    name: 'Custom',
    preset: 'default',
    colors: {},
    layout: {},
    raw: '',
    scheme: '',
  }

  function clone(o) {
    return JSON.parse(JSON.stringify(o))
  }

  function normalizeState(raw) {
    var s = clone(DEFAULT_STATE)
    if (raw && typeof raw === 'object') {
      if (raw.name) s.name = String(raw.name)
      if (raw.preset) s.preset = String(raw.preset)
      if (raw.colors && typeof raw.colors === 'object') s.colors = raw.colors
      if (raw.layout && typeof raw.layout === 'object') s.layout = raw.layout
      if (typeof raw.raw === 'string') s.raw = raw.raw
      if (raw.scheme) s.scheme = String(raw.scheme)
    }
    return s
  }

  function readCookie(name) {
    var m = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '=([^;]*)'))
    return m ? decodeURIComponent(m[1]) : null
  }

  function writeCookie(name, value, days) {
    var exp = new Date(Date.now() + days * 86400000).toUTCString()
    document.cookie = name + '=' + encodeURIComponent(value) + '; expires=' + exp + '; path=/; SameSite=Lax'
  }

  function eraseCookie(name) {
    document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; SameSite=Lax'
  }

  function saveToCookies(payload) {
    try {
      // clear previous chunks
      var prev = parseInt(readCookie(COOKIE_COUNT) || '0', 10)
      for (var i = 0; i < prev; i++) eraseCookie(COOKIE_PREFIX + '_' + i)
      eraseCookie(COOKIE_PREFIX)
      var chunks = []
      for (var p = 0; p < payload.length; p += COOKIE_CHUNK) chunks.push(payload.slice(p, p + COOKIE_CHUNK))
      if (chunks.length > 12) return false
      chunks.forEach(function (c, idx) {
        writeCookie(COOKIE_PREFIX + '_' + idx, c, COOKIE_DAYS)
      })
      writeCookie(COOKIE_COUNT, String(chunks.length), COOKIE_DAYS)
      return true
    } catch (e) {
      return false
    }
  }

  function loadFromCookies() {
    try {
      var n = parseInt(readCookie(COOKIE_COUNT) || '0', 10)
      if (n > 0) {
        var parts = []
        for (var i = 0; i < n; i++) {
          var c = readCookie(COOKIE_PREFIX + '_' + i)
          if (c === null) return null
          parts.push(c)
        }
        return JSON.parse(parts.join(''))
      }
      var single = readCookie(COOKIE_PREFIX)
      if (single) return JSON.parse(single)
    } catch (e) {}
    return null
  }

  function loadState() {
    var fromCookie = loadFromCookies()
    if (fromCookie) return normalizeState(fromCookie)
    try {
      var ls = localStorage.getItem(LS_KEY)
      if (ls) {
        var parsed = normalizeState(JSON.parse(ls))
        // migrate the same-session cache into cookies for the next launch
        saveState(parsed)
        return parsed
      }
    } catch (e) {}
    // A theme baked in at install time by the injector (--theme).
    if (window.__FREEBUFF_THEME_DEFAULT__) return normalizeState(window.__FREEBUFF_THEME_DEFAULT__)
    return clone(DEFAULT_STATE)
  }

  var saveTimer = null
  function saveState(state, immediate) {
    var payload = JSON.stringify(state)
    function commit() {
      try {
        localStorage.setItem(LS_KEY, payload)
      } catch (e) {}
      saveToCookies(payload)
    }
    if (immediate) return commit()
    clearTimeout(saveTimer)
    saveTimer = setTimeout(commit, 250)
  }

  /* ------------------------------------------------------------------ *
   * Applying the theme
   * ------------------------------------------------------------------ */

  var state = loadState()
  var applied = { colors: {}, layout: {} }

  function setVar(name, value) {
    document.documentElement.style.setProperty(name, value)
  }

  function unsetVar(name) {
    document.documentElement.style.removeProperty(name)
  }

  function currentValues() {
    // Order matters: preset is the base, explicit overrides win on top.
    var out = {}
    var preset = state.preset !== 'default' ? PRESET_BY_ID[state.preset] : null
    if (preset && preset.colors) for (var k0 in preset.colors) out[k0] = preset.colors[k0]
    for (var k in state.colors) out[k] = toCss(state.colors[k])
    for (var k2 in state.layout) out[k2] = state.layout[k2]
    return out
  }

  function applyState() {
    var next = currentValues()
    var name
    for (name in applied.colors) if (!(name in next)) unsetVar(name)
    for (name in applied.layout) if (!(name in next)) unsetVar(name)
    applied.colors = {}
    applied.layout = {}
    for (name in next) {
      setVar(name, next[name])
      if (next[name] && typeof next[name] === 'object') applied.colors[name] = true
      else applied.layout[name] = true
    }
    // color-scheme: preset appearance, unless the user forced one
    if (state.scheme === 'light' || state.scheme === 'dark') {
      document.documentElement.style.setProperty('color-scheme', state.scheme)
    } else {
      var preset2 = state.preset !== 'default' ? PRESET_BY_ID[state.preset] : null
      if (preset2 && preset2.scheme) document.documentElement.style.setProperty('color-scheme', preset2.scheme)
      else document.documentElement.style.removeProperty('color-scheme')
    }
    applyRaw()
  }

  function applyRaw() {
    var el = document.getElementById(RAW_STYLE_ID)
    if (!state.raw) {
      if (el) el.remove()
      return
    }
    if (!el) {
      el = document.createElement('style')
      el.id = RAW_STYLE_ID
      document.head.appendChild(el)
    }
    el.textContent = state.raw
  }

  function resetAll() {
    state = clone(DEFAULT_STATE)
    var name
    for (name in applied.colors) unsetVar(name)
    for (name in applied.layout) unsetVar(name)
    applied = { colors: {}, layout: {} }
    document.documentElement.style.removeProperty('color-scheme')
    applyRaw()
    saveState(state, true)
  }

  function cssText() {
    var lines = [':root {']
    var values = currentValues()
    for (var k in values) lines.push('  ' + k + ': ' + values[k] + ';')
    lines.push('}')
    if (state.raw) lines.push('', state.raw)
    return lines.join('\n')
  }

  // Apply before the UI mounts so there is no flash of the default theme.
  applyState()

  /* ------------------------------------------------------------------ *
   * UI
   * ------------------------------------------------------------------ */

  // Set by mount(): lets the sidebar button drive the page.
  var ui = null
  var railButton = null
  var pagePanel = null

  var RAIL_ICON =
    '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" ' +
    'stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
    '<path d="M12 3a9 9 0 1 0 0 18c1.1 0 1.9-.8 1.9-1.8 0-.5-.2-.9-.5-1.2-.3-.3-.5-.7-.5-1.1 0-.9.8-1.6 1.7-1.6h1.1A5.3 5.3 0 0 0 21 9.9C21 6 16.97 3 12 3Z"/>' +
    '<circle cx="7.7" cy="12" r="1.05"/><circle cx="10" cy="7.9" r="1.05"/>' +
    '<circle cx="14.3" cy="7.9" r="1.05"/><circle cx="17" cy="11.6" r="1.05"/></svg>'

  /**
   * Freebuff's icon rail lives in `.shell-navigation-top` as `.shell-nav-button`
   * elements. Injecting one there means the shell styles it for us - we only
   * supply the icon and the handler.
   */
  function installRailButton() {
    if (railButton && railButton.isConnected) return true
    var rail =
      document.querySelector('.shell-navigation-top') ||
      document.querySelector('.shell-navigation')
    if (!rail) return false
    var btn = document.createElement('button')
    btn.className = 'shell-nav-button'
    btn.setAttribute('data-fbts-rail', '1')
    btn.setAttribute('aria-label', 'Theme Studio')
    btn.setAttribute('title', 'Theme Studio')
    btn.innerHTML = RAIL_ICON
    btn.addEventListener('click', function (e) {
      e.preventDefault()
      e.stopPropagation()
      if (ui) ui.toggle()
    })
    rail.appendChild(btn)
    railButton = btn
    return true
  }

  function watchRail() {
    var scheduled = false
    var recheck = function () {
      if (scheduled) return
      scheduled = true
      requestAnimationFrame(function () {
        scheduled = false
        installRailButton()
        // The shell may have re-laid itself out; keep the page on the frame.
        fitPage()
      })
    }
    try {
      new MutationObserver(recheck).observe(document.body, { childList: true, subtree: true })
    } catch (e) {}
    // Belt and braces: React can replace the rail wholesale on navigation.
    setInterval(installRailButton, 1500)
  }

  /*
   * Where the page belongs.
   *
   * Freebuff renders its content inside `.workspace-frame`, which the shell
   * insets by --shell-rail-width on the left and --shell-inset on the right and
   * bottom, below a --tabbar-height tab row. Those three custom properties are
   * declared on `.desktop-shell`, NOT on :root, and our shadow host is a child
   * of <body> - so `var(--shell-rail-width)` inside the shadow root silently
   * took its fallback (56px instead of 52px, 60px instead of 48px for the tab
   * bar) and the page never lined up with the app's own views.
   *
   * Measuring the frame instead of recomputing the arithmetic also tracks
   * compact mode, a collapsed sidebar, thread windows and future shell tweaks.
   */
  /** The biggest visible workspace box on the page (there is normally one). */
  function workspaceBox() {
    var best = null
    var nodes = document.querySelectorAll('.workspace-frame, .settings-frame')
    for (var i = 0; i < nodes.length; i++) {
      var r = nodes[i].getBoundingClientRect()
      if (r.width < 240 || r.height < 160) continue
      if (!best || r.width * r.height > best.width * best.height)
        best = { width: r.width, height: r.height, top: r.top, left: r.left, right: r.right, bottom: r.bottom }
    }
    return best
  }

  function measureWorkspace() {
    var frame = workspaceBox()
    if (frame) return frame
    var shell = document.querySelector('.desktop-shell') || document.documentElement
    var cs = getComputedStyle(shell)
    var num = function (name, fallback) {
      var v = parseFloat(cs.getPropertyValue(name))
      return isFinite(v) ? v : fallback
    }
    var box = shell.getBoundingClientRect()
    var bar = num('--tabbar-height', 48)
    var rail = num('--shell-rail-width', 52)
    var inset = num('--shell-inset', 8)
    return {
      top: box.top + bar,
      left: box.left + rail,
      right: box.right - inset,
      bottom: box.bottom - inset,
      corner: cs.getPropertyValue('--workspace-corner').trim(),
    }
  }

  function fitToWorkspace(node) {
    if (!node || !node.isConnected) return
    var box = measureWorkspace()
    var corner = box.corner
    if (!corner) {
      var shell = document.querySelector('.desktop-shell')
      if (shell) {
        try {
          corner = getComputedStyle(shell).getPropertyValue('--workspace-corner').trim()
        } catch (e) {}
      }
    }
    node.style.top = Math.round(box.top) + 'px'
    node.style.left = Math.round(box.left) + 'px'
    node.style.right = Math.max(0, Math.round(window.innerWidth - box.right)) + 'px'
    node.style.bottom = Math.max(0, Math.round(window.innerHeight - box.bottom)) + 'px'
    // Match the workspace frame's rounded corners, whatever the shell uses.
    node.style.borderRadius = corner || '18px'
  }

  function fitPage() {
    fitToWorkspace(pagePanel)
  }

  /** Clicking any other rail icon means the user left our page. */
  function closePageOnExternalNav() {
    document.addEventListener(
      'click',
      function (e) {
        var t = e.target
        var btn = t && t.closest ? t.closest('.shell-nav-button') : null
        if (!btn || btn === railButton) return
        if (ui) ui.toggle(false)
      },
      true,
    )
  }

  function el(tag, props, children) {
    var node = document.createElement(tag)
    if (props)
      for (var k in props) {
        if (k === 'class') node.className = props[k]
        else if (k === 'text') node.textContent = props[k]
        else if (k === 'html') node.innerHTML = props[k]
        else if (k.indexOf('on') === 0 && typeof props[k] === 'function') node.addEventListener(k.slice(2), props[k])
        else if (k === 'style') node.style.cssText = props[k]
        else if (props[k] != null) node.setAttribute(k, props[k])
      }
    ;(children || []).forEach(function (c) {
      if (c == null) return
      node.appendChild(typeof c === 'string' ? document.createTextNode(c) : c)
    })
    return node
  }

  var STYLES = `
* { box-sizing: border-box; margin: 0; padding: 0; font-family: var(--fbts-font); }
:host {
  --fbts-font: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
  --fbts-mono: ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace;
  --fbts-bg: #121317;
  --fbts-panel: #17181d;
  --fbts-panel2: #1d1f25;
  --fbts-line: #2a2d35;
  --fbts-ink: #e7e9ee;
  --fbts-mute: #9aa1ad;
  --fbts-faint: #6d7480;
  --fbts-accent: #5b8cff;
  --fbts-accent2: #8affc1;
  --fbts-danger: #ff6b6b;
  --fbts-radius: 12px;
}
.fbts-root { position: fixed; inset: 0; pointer-events: none; z-index: 2147483600; }

.fbts-panel {
  position: fixed; right: 18px; bottom: 74px; width: 384px; max-width: calc(100vw - 36px);
  max-height: min(78vh, 780px); pointer-events: auto; display: none; flex-direction: column;
  background: var(--fbts-panel); border: 1px solid var(--fbts-line);
  border-radius: 16px; overflow: hidden; color: var(--fbts-ink);
  box-shadow: 0 30px 90px rgba(0,0,0,.68), 0 0 0 1px rgba(255,255,255,.03) inset;
}
.fbts-panel.open { display: flex; }

.fbts-head { display: flex; align-items: center; gap: 10px; padding: 12px 12px 10px 14px; border-bottom: 1px solid var(--fbts-line); background: var(--fbts-panel2); }
.fbts-title { font-size: 13px; font-weight: 700; letter-spacing: .01em; }
.fbts-head-row { display: flex; align-items: center; gap: 8px; }
.fbts-badge {
  display: inline-flex; align-items: center; height: 17px; padding: 0 7px; flex: none;
  border: 1px solid color-mix(in srgb, var(--fbts-ink) 20%, transparent); border-radius: 999px;
  color: var(--fbts-mute); font-size: 9px; font-weight: 700; letter-spacing: .07em; text-transform: uppercase;
}
.fbts-sub { font-size: 10.5px; color: var(--fbts-faint); font-weight: 500; }
.fbts-sub-sep { opacity: .5; padding: 0 3px; }
.fbts-version { cursor: pointer; }
.fbts-version:hover { color: var(--fbts-mute); text-decoration: underline; }
.fbts-head .grow { flex: 1; }
.fbts-x { width: 26px; height: 26px; border-radius: 8px; border: 1px solid var(--fbts-line); background: transparent; color: var(--fbts-mute); cursor: pointer; font-size: 14px; line-height: 1; }
.fbts-x:hover { color: var(--fbts-ink); border-color: var(--fbts-accent); }

.fbts-tabs { display: flex; gap: 2px; padding: 8px 8px 0; border-bottom: 1px solid var(--fbts-line); background: var(--fbts-panel2); overflow-x: auto; scrollbar-width: none; }
.fbts-tabs::-webkit-scrollbar { display: none; }
.fbts-tab { padding: 7px 11px; border-radius: 8px 8px 0 0; font-size: 11.5px; font-weight: 600; color: var(--fbts-mute); cursor: pointer; border: 1px solid transparent; border-bottom: none; white-space: nowrap; }
.fbts-tab:hover { color: var(--fbts-ink); }
.fbts-tab.active { color: var(--fbts-ink); background: var(--fbts-panel); border-color: var(--fbts-line); }

.fbts-body { overflow-y: auto; padding: 12px; flex: 1; }
.fbts-body::-webkit-scrollbar { width: 10px; }
.fbts-body::-webkit-scrollbar-thumb { background: #31353f; border-radius: 6px; border: 3px solid var(--fbts-panel); }

.fbts-section { margin-bottom: 16px; }
.fbts-section-title { font-size: 10px; text-transform: uppercase; letter-spacing: .09em; color: var(--fbts-faint); font-weight: 700; margin: 0 0 8px 2px; }

.fbts-presets { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.fbts-preset { border: 1px solid var(--fbts-line); border-radius: 10px; padding: 8px; cursor: pointer; background: var(--fbts-panel2); text-align: left; transition: border-color .12s, transform .12s; }
.fbts-preset:hover { border-color: var(--fbts-accent); transform: translateY(-1px); }
.fbts-preset.active { border-color: var(--fbts-accent); box-shadow: 0 0 0 2px rgba(91,140,255,.22); }
.fbts-swatches { display: flex; gap: 3px; margin-bottom: 6px; }
.fbts-swatch { width: 100%; height: 16px; border-radius: 4px; border: 1px solid rgba(255,255,255,.09); }
.fbts-preset-name { font-size: 11px; font-weight: 600; color: var(--fbts-ink); }

.fbts-row { display: flex; align-items: center; gap: 8px; padding: 4px 0; }
.fbts-row-label { flex: 1; font-size: 11.5px; color: var(--fbts-mute); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.fbts-row-label code { font-family: var(--fbts-mono); font-size: 9.5px; color: var(--fbts-faint); display: block; }
.fbts-swatchinput { width: 26px; height: 24px; padding: 0; border: 1px solid var(--fbts-line); border-radius: 6px; background: none; cursor: pointer; flex: none; }
.fbts-swatchinput::-webkit-color-swatch-wrapper { padding: 2px; }
.fbts-swatchinput::-webkit-color-swatch { border: none; border-radius: 4px; }
.fbts-textinput {
  width: 104px; flex: none; background: var(--fbts-panel2); border: 1px solid var(--fbts-line); color: var(--fbts-ink);
  border-radius: 7px; padding: 5px 7px; font-size: 11px; font-family: var(--fbts-mono); outline: none;
}
.fbts-textinput:focus { border-color: var(--fbts-accent); }
.fbts-alpha { width: 52px; flex: none; accent-color: var(--fbts-accent); }
.fbts-mini { width: 24px; height: 24px; flex: none; border-radius: 6px; border: 1px solid var(--fbts-line); background: transparent; color: var(--fbts-faint); cursor: pointer; font-size: 11px; }
.fbts-mini:hover { color: var(--fbts-danger); border-color: var(--fbts-danger); }

.fbts-accordion { border: 1px solid var(--fbts-line); border-radius: 10px; margin-bottom: 8px; overflow: hidden; background: var(--fbts-panel2); }
.fbts-accordion-head { display: flex; align-items: center; gap: 8px; padding: 9px 11px; cursor: pointer; font-size: 11.5px; font-weight: 600; user-select: none; }
.fbts-accordion-head .chev { color: var(--fbts-faint); font-size: 10px; transition: transform .15s; }
.fbts-accordion.open .chev { transform: rotate(90deg); }
.fbts-accordion-body { display: none; padding: 4px 11px 10px; border-top: 1px solid var(--fbts-line); }
.fbts-accordion.open .fbts-accordion-body { display: block; }

.fbts-btn { padding: 7px 12px; border-radius: 8px; border: 1px solid var(--fbts-line); background: var(--fbts-panel2); color: var(--fbts-ink); font-size: 11.5px; font-weight: 600; cursor: pointer; }
.fbts-btn:hover { border-color: var(--fbts-accent); }
.fbts-btn.primary { background: var(--fbts-accent); border-color: var(--fbts-accent); color: #0a0c10; }
.fbts-btn.danger:hover { border-color: var(--fbts-danger); color: var(--fbts-danger); }
.fbts-actions { display: flex; flex-wrap: wrap; gap: 7px; padding: 10px 12px; border-top: 1px solid var(--fbts-line); background: var(--fbts-panel2); }
.fbts-actions .grow { flex: 1; }

.fbts-note { font-size: 10.5px; color: var(--fbts-faint); line-height: 1.5; margin: 6px 2px 0; }
.fbts-textarea { width: 100%; min-height: 150px; background: #0e0f13; border: 1px solid var(--fbts-line); border-radius: 9px; color: var(--fbts-ink); font-family: var(--fbts-mono); font-size: 11px; line-height: 1.5; padding: 9px; resize: vertical; outline: none; }
.fbts-textarea:focus { border-color: var(--fbts-accent); }
.fbts-search { width: 100%; background: var(--fbts-panel2); border: 1px solid var(--fbts-line); border-radius: 8px; color: var(--fbts-ink); padding: 7px 9px; font-size: 11.5px; outline: none; margin-bottom: 8px; }
.fbts-search:focus { border-color: var(--fbts-accent); }
.fbts-toast {
  position: fixed; right: 18px; bottom: 18px; pointer-events: none; opacity: 0;
  background: #0e1015; border: 1px solid var(--fbts-line); color: var(--fbts-ink);
  padding: 9px 14px; border-radius: 10px; font-size: 11.5px; font-weight: 600;
  transition: opacity .2s, transform .2s; transform: translateY(6px);
  box-shadow: 0 12px 34px rgba(0,0,0,.6);
}
.fbts-toast.show { opacity: 1; transform: translateY(0); }

/* ---- update chip + popup ---- */
.fbts-update-chip {
  position: fixed; right: 16px; bottom: 16px; z-index: 2; pointer-events: auto; display: none;
  align-items: center; gap: 7px; padding: 8px 13px 8px 10px;
  border: 1px solid var(--fbts-line); border-radius: 999px; background: var(--fbts-panel2);
  color: var(--fbts-ink); font-size: 11.5px; font-weight: 600; cursor: pointer; white-space: nowrap;
  box-shadow: 0 10px 30px rgba(0,0,0,.5);
}
.fbts-update-chip.show { display: inline-flex; }
.fbts-update-chip:hover { border-color: var(--fbts-accent); }
.fbts-update-chip svg { width: 15px; height: 15px; flex: none; color: var(--fbts-accent); }
/* Keep clear of the page footer while Theme Studio is open. */
.fbts-root.fbts-page-open .fbts-update-chip { right: 26px; bottom: 68px; }

.fbts-modal-wrap {
  position: fixed; inset: 0; z-index: 3; display: none; place-items: center; padding: 24px;
  pointer-events: auto; background: rgba(0,0,0,.55);
}
.fbts-modal-wrap.show { display: grid; }
.fbts-modal {
  width: min(430px, calc(100vw - 48px)); padding: 18px 20px 16px;
  border: 1px solid var(--fbts-line); border-radius: 16px; background: var(--fbts-panel);
  color: var(--fbts-ink); box-shadow: 0 30px 90px rgba(0,0,0,.6);
}
.fbts-modal h3 { font-size: 15px; font-weight: 700; margin-bottom: 8px; }
.fbts-modal p { font-size: 11.5px; line-height: 1.6; color: var(--fbts-mute); margin-bottom: 9px; }
.fbts-modal p:last-of-type { margin-bottom: 0; }
.fbts-modal code { font-family: var(--fbts-mono); color: var(--fbts-ink); }
.fbts-modal-match { margin: 12px 2px 0; font-size: 10.5px; color: var(--fbts-faint); }
.fbts-modal-foot { display: flex; gap: 8px; align-items: center; margin-top: 14px; }
.fbts-modal-foot .grow { flex: 1; }
`

  /*
   * The panel is really a page: it fills the workspace to the right of
   * Freebuff's icon rail and sits below the tab bar, exactly where the app
   * renders its own settings view. It is themed with the live app tokens, so
   * the studio follows whatever theme is currently applied.
   */
  var PAGE_STYLES = `
:host {
  --fbts-font: var(--font-sans, ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif);
  --fbts-mono: var(--font-mono, ui-monospace, Menlo, Consolas, monospace);
  --fbts-bg: var(--bg, #16171b);
  --fbts-panel: var(--bg, #16171b);
  --fbts-panel2: var(--chrome, var(--surface, #1d1f24));
  --fbts-line: var(--border, #2a2d35);
  --fbts-ink: var(--text, #e7e9ee);
  --fbts-mute: var(--muted, #9aa1ad);
  --fbts-faint: var(--faint, #6d7480);
  --fbts-accent: var(--brand, #5b8cff);
  --fbts-accent2: var(--brand-dim, #8affc1);
  --fbts-danger: var(--danger, #ff6b6b);
  --fbts-radius: var(--radius-md, 10px);
  --fbts-body-size: var(--font-size-body, 13px);
  --fbts-ui-size: var(--font-size-ui, 12px);
  --fbts-label-size: var(--font-size-label, 11px);
}

.fbts-panel {
  position: fixed;
  z-index: 1;
  /*
   * Fallbacks only. The shell defines --tabbar-height / --shell-rail-width /
   * --shell-inset on .desktop-shell, not on :root, and this shadow host is a
   * child of <body> - so it cannot inherit them and var() here would silently
   * fall back. fitToWorkspace() measures the real workspace frame instead and
   * writes the box as inline styles.
   */
  top: 48px;
  left: 52px;
  right: 8px;
  bottom: 8px;
  width: auto;
  max-width: none;
  max-height: none;
  border: none;
  border-radius: 18px;
  box-shadow: none;
  overflow: hidden;
  background: var(--fbts-bg);
}

.fbts-head {
  padding: 18px 24px 14px;
  background: transparent;
  border-bottom: 1px solid var(--fbts-line);
}
.fbts-title { font-size: var(--font-size-heading, 18px); font-weight: var(--font-weight-semibold, 600); }
.fbts-sub { font-size: var(--fbts-label-size); }
.fbts-x {
  width: auto; height: 30px; padding: 0 12px; font-size: var(--fbts-ui-size);
  font-weight: var(--font-weight-medium, 450);
}
.fbts-x:hover { background: color-mix(in srgb, var(--fbts-ink) 8%, transparent); }

.fbts-page-main { flex: 1; display: flex; min-height: 0; }

.fbts-tabs {
  flex-direction: column;
  align-items: stretch;
  gap: 2px;
  width: 200px;
  flex: none;
  padding: 14px 12px;
  border-bottom: none;
  border-right: 1px solid var(--fbts-line);
  background: var(--fbts-panel2);
  overflow-x: visible;
  overflow-y: auto;
}
.fbts-tab {
  padding: 8px 11px;
  border: none;
  border-radius: var(--fbts-radius);
  font-size: var(--fbts-ui-size);
  font-weight: var(--font-weight-medium, 450);
}
.fbts-tab.active { border: none; background: color-mix(in srgb, var(--fbts-ink) 11%, transparent); }
.fbts-tab:hover:not(.active) { background: color-mix(in srgb, var(--fbts-ink) 6%, transparent); }

.fbts-body { padding: 22px 28px 40px; }
.fbts-actions { background: transparent; padding: 14px 24px; }
.fbts-presets { grid-template-columns: repeat(auto-fill, minmax(148px, 1fr)); gap: 10px; }
.fbts-preset { padding: 10px; border-radius: var(--fbts-radius); }
.fbts-accordion { border-radius: var(--fbts-radius); }
.fbts-row-label { font-size: var(--fbts-ui-size); }
.fbts-textinput { width: 128px; border-radius: var(--fbts-radius); }
.fbts-swatchinput { border-radius: var(--fbts-radius); }
.fbts-toast { left: 50%; right: auto; bottom: 28px; transform: translate(-50%, 8px); }
.fbts-toast.show { transform: translate(-50%, 0); }
`

  function mount() {
    if (document.querySelector('[data-fbts-host]')) return

    var host = el('div', { 'data-fbts-host': '' })
    // Deliberately not `all: initial` - that would also reset the --fbts-* tokens
    // the shadow stylesheet inherits from :host.
    host.style.cssText = 'position: fixed; z-index: 2147483600; right: 0; bottom: 0; width: 0; height: 0; pointer-events: none;'
    document.body.appendChild(host)
    var shadow = host.attachShadow({ mode: 'open' })
    shadow.appendChild(el('style', { text: STYLES }))
    shadow.appendChild(el('style', { text: PAGE_STYLES }))

    var panel = el('div', { class: 'fbts-panel' })

    var toast = el('div', { class: 'fbts-toast' })
    var toastTimer = null
    function showToast(msg) {
      toast.textContent = msg
      toast.classList.add('show')
      clearTimeout(toastTimer)
      toastTimer = setTimeout(function () {
        toast.classList.remove('show')
      }, 1900)
    }

    /* ---- update chip + popup ---- */
    var UPDATE_ICON =
      '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" ' +
      'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
      '<path d="M12 3.5v11"/><path d="M7.5 10.5 12 15l4.5-4.5"/><path d="M4.5 20h15"/></svg>'

    var chipLabel = el('span', { text: 'Update' })
    var chip = el(
      'button',
      {
        class: 'fbts-update-chip',
        type: 'button',
        title: 'A newer version of Freebuff Theme Studio is available',
        onclick: function () {
          showUpdatePopup()
        },
      },
      [el('span', { html: UPDATE_ICON }), chipLabel],
    )

    var modalBody = el('div', { class: 'fbts-modal' })
    var modalWrap = el('div', { class: 'fbts-modal-wrap', onclick: function (e) { if (e.target === modalWrap) closePopup() } }, [modalBody])

    var remoteVersion = ''
    var checking = false

    function showChip(version) {
      chipLabel.textContent = 'Update ' + version
      chip.classList.add('show')
    }
    function hideChip() {
      chip.classList.remove('show')
    }
    function closePopup() {
      modalWrap.classList.remove('show')
    }

    function openRelease(version) {
      // Freebuff's main process denies off-origin popups and hands the URL to
      // the system browser, so this never opens a second app window.
      try {
        window.open(RELEASES_URL + version, '_blank', 'noopener')
      } catch (e) {}
    }

    function showUpdatePopup() {
      if (!remoteVersion) return
      modalBody.textContent = ''
      modalBody.appendChild(el('h3', { text: 'Update available' }))
      modalBody.appendChild(
        el('p', {}, [
          'Freebuff Theme Studio ',
          el('code', { text: 'v' + remoteVersion }),
          ' is out. You have ',
          el('code', { text: 'v' + VERSION }),
          '.',
        ]),
      )
      modalBody.appendChild(
        el('p', { text: 'Download the new installer from GitHub and run it the same way you installed this one. Your theme is kept.' }),
      )
      modalBody.appendChild(
        el('p', { text: 'Update opens the release page in your browser - nothing is downloaded automatically. Press Esc to decide later.' }),
      )
      modalBody.appendChild(el('p', { class: 'fbts-modal-match', text: 'Unofficial extension - not made by Freebuff.' }))
      modalBody.appendChild(
        el('div', { class: 'fbts-modal-foot' }, [
          el('button', {
            class: 'fbts-btn primary',
            type: 'button',
            text: 'Update',
            onclick: function () {
              openRelease(remoteVersion)
              closePopup()
              hideChip()
              showToast('Opening the release page')
            },
          }),
          el('button', {
            class: 'fbts-btn',
            type: 'button',
            text: 'Skip this version',
            onclick: function () {
              writeCookie(SKIP_COOKIE, remoteVersion, 365)
              closePopup()
              showChip(remoteVersion)
              showToast('Update button moved to the corner')
            },
          }),
        ]),
      )
      modalWrap.classList.add('show')
    }

    function isNewer(a, b) {
      var pa = String(a).split('.')
      var pb = String(b).split('.')
      if (pa.length < 2) return false
      for (var i = 0; i < Math.max(pa.length, pb.length); i++) {
        var na = parseInt(pa[i], 10) || 0
        var nb = parseInt(pb[i], 10) || 0
        if (na !== nb) return na > nb
      }
      return false
    }

    var cleanVersion = function (v) {
      return String(v == null ? '' : v).replace(/["'\s]/g, '')
    }

    /**
     * The release feed is a stylesheet that only declares
     * `--fbts-remote-version`. Reading it back through getComputedStyle works
     * even cross-origin, and a missing or blocked feed just means silence.
     */
    function checkForUpdates(manual) {
      if (checking || !document.head) return
      var last = parseInt(readCookie(CHECK_COOKIE) || '0', 10)
      if (!manual && Date.now() - last < CHECK_INTERVAL) return
      checking = true
      if (manual) showToast('Checking for updates\u2026')

      var link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = UPDATE_FEED
      var settled = false

      function settle(version) {
        if (settled) return
        settled = true
        checking = false
        if (link.parentNode) link.parentNode.removeChild(link)
        document.documentElement.removeAttribute(UPDATE_PROBE)
        if (!version) {
          if (manual) showToast('Could not check for updates')
          return
        }
        writeCookie(CHECK_COOKIE, String(Date.now()), 365)
        if (!isNewer(version, VERSION)) {
          if (manual) showToast('You are on the latest version')
          return
        }
        remoteVersion = version
        // A skipped version stays skipped: it only leaves the corner button.
        if (!manual && readCookie(SKIP_COOKIE) === version) {
          showChip(version)
          return
        }
        showUpdatePopup()
      }

      link.addEventListener('load', function () {
        var value = ''
        try {
          value = getComputedStyle(document.documentElement).getPropertyValue(REMOTE_VERSION_VAR)
        } catch (e) {}
        settle(cleanVersion(value))
      })
      link.addEventListener('error', function () {
        settle('')
      })
      setTimeout(function () {
        settle('')
      }, CHECK_TIMEOUT)

      document.documentElement.setAttribute(UPDATE_PROBE, '')
      document.head.appendChild(link)
    }

    var root = el('div', { class: 'fbts-root' }, [panel, toast, chip, modalWrap])
    shadow.appendChild(root)

    /* ---- header ---- */
    var schemeLabel = el('span', {
      class: 'fbts-version',
      title: 'Check for updates',
      text: 'v' + VERSION,
      onclick: function () { checkForUpdates(true) },
    })
    var head = el('div', { class: 'fbts-head' }, [
      el('div', {}, [
        el('div', { class: 'fbts-head-row' }, [
          el('div', { class: 'fbts-title', text: 'Theme Studio' }),
          el('span', {
            class: 'fbts-badge',
            title: 'A community extension. Not made by Freebuff.',
            text: 'Unofficial',
          }),
        ]),
        el('div', { class: 'fbts-sub' }, [
          schemeLabel,
          el('span', { class: 'fbts-sub-sep', text: '\u00b7' }),
          el('span', { text: 'not made by Freebuff' }),
        ]),
      ]),
      el('div', { class: 'grow' }),
      el('button', { class: 'fbts-x', title: 'Close', text: '\u00d7', onclick: function () { panel.classList.remove('open') } }),
    ])

    /* ---- tabs ---- */
    var TAB_DEFS = [
      ['presets', 'Presets'],
      ['colors', 'Colors'],
      ['layout', 'Layout'],
      ['advanced', 'Advanced'],
      ['io', 'Export'],
    ]
    var tabsBar = el('div', { class: 'fbts-tabs' })
    var body = el('div', { class: 'fbts-body' })
    var tabPanes = {}

    TAB_DEFS.forEach(function (def) {
      var tab = el('div', { class: 'fbts-tab', text: def[1], onclick: function () { selectTab(def[0]) } })
      tab.dataset.tab = def[0]
      tabsBar.appendChild(tab)
    })

    function selectTab(id) {
      Array.prototype.forEach.call(tabsBar.children, function (c) {
        c.classList.toggle('active', c.dataset.tab === id)
      })
      for (var k in tabPanes) tabPanes[k].style.display = k === id ? '' : 'none'
    }

    function pane(id) {
      var p = el('div')
      p.dataset.pane = id
      tabPanes[id] = p
      body.appendChild(p)
      return p
    }

    /* ---- persistence helpers ---- */
    function commit(message) {
      applyState()
      saveState(state)
      refreshActive()
      if (message) showToast(message)
    }

    function setOverride(token, value) {
      if (value == null || value === '') {
        delete state.colors[token]
        delete state.layout[token]
      } else if (typeof value === 'object') {
        state.colors[token] = value
        delete state.layout[token]
      } else {
        state.layout[token] = value
        delete state.colors[token]
      }
      commit()
    }

    function currentColorFor(token) {
      if (state.colors[token]) {
        var c = state.colors[token]
        return typeof c === 'string' ? parseColor(c) || { hex: '#000000', a: 1 } : { hex: c.hex, a: c.a == null ? 1 : c.a }
      }
      return resolveTokenColor(token) || { hex: '#808080', a: 1 }
    }

    /* ---- color row ---- */
    function colorRow(token, labelText) {
      var initial = currentColorFor(token)
      var colorInput = el('input', { class: 'fbts-swatchinput', type: 'color', value: initial.hex })
      var alpha = el('input', { class: 'fbts-alpha', type: 'range', min: '0', max: '100', step: '1', value: String(Math.round(initial.a * 100)) })
      var hex = el('input', { class: 'fbts-textinput', type: 'text', value: initial.hex, spellcheck: 'false' })
      var reset = el('button', { class: 'fbts-mini', title: 'Reset', text: '\u21ba' })

      function push(commitNow) {
        var parsed = parseColor(colorInput.value) || { hex: '#000000', a: 1 }
        var a = parseFloat(alpha.value) / 100
        if (commitNow) {
          setOverride(token, { hex: parsed.hex, a: a })
        }
      }

      colorInput.addEventListener('input', function () {
        hex.value = colorInput.value
        push(true)
      })
      hex.addEventListener('input', function () {
        var parsed = parseColor(hex.value)
        if (parsed) {
          colorInput.value = parsed.hex
          push(true)
        }
      })
      alpha.addEventListener('input', function () { push(true) })
      reset.addEventListener('click', function () {
        setOverride(token, null)
        var fresh = resolveTokenColor(token) || { hex: '#808080', a: 1 }
        colorInput.value = fresh.hex
        hex.value = fresh.hex
        alpha.value = String(Math.round(fresh.a * 100))
        showToast('Reset ' + token)
      })

      var row = el('div', { class: 'fbts-row', 'data-token': token }, [
        el('div', { class: 'fbts-row-label', title: token }, [
          el('span', { text: labelText || token }),
          el('code', { text: token }),
        ]),
        colorInput,
        alpha,
        hex,
        reset,
      ])
      return row
    }

    /* ---- text/size row ---- */
    function textRow(token, labelText) {
      var current = state.layout[token] || ''
      var input = el('input', {
        class: 'fbts-textinput', type: 'text', value: current, placeholder: 'app default', spellcheck: 'false',
        oninput: function () { setOverride(token, input.value.trim() || null) },
      })
      var reset = el('button', {
        class: 'fbts-mini', title: 'Reset', text: '\u21ba',
        onclick: function () {
          input.value = ''
          setOverride(token, null)
          showToast('Reset ' + token)
        },
      })
      return el('div', { class: 'fbts-row', 'data-token': token }, [
        el('div', { class: 'fbts-row-label', title: token }, [
          el('span', { text: labelText || token }),
          el('code', { text: token }),
        ]),
        input,
        reset,
      ])
    }

    /* ---- Presets tab ---- */
    var presetsPane = pane('presets')
    var presetsGrid = el('div', { class: 'fbts-presets' })
    PRESETS.forEach(function (p) {
      var swatchColors = presetSwatches(p)
      var card = el(
        'div',
        {
          class: 'fbts-preset', 'data-preset': p.id,
          onclick: function () {
            state.preset = p.id
            state.colors = {}
            commit('Applied ' + p.label)
          },
        },
        [
          el('div', { class: 'fbts-swatches' }, swatchColors.map(function (c) {
            var s = el('span', { class: 'fbts-swatch' })
            s.style.background = c
            return s
          })),
          el('div', { class: 'fbts-preset-name', text: p.label }),
        ],
      )
      presetsGrid.appendChild(card)
    })
    presetsPane.appendChild(el('div', { class: 'fbts-section' }, [
      el('div', { class: 'fbts-section-title', text: 'Presets' }),
      presetsGrid,
    ]))
    var schemeRow = el('div', { class: 'fbts-row' }, [
      el('div', { class: 'fbts-row-label', text: 'Color scheme' }),
    ])
    ;['auto', 'dark', 'light'].forEach(function (mode) {
      schemeRow.appendChild(
        el('button', {
          class: 'fbts-btn', text: mode, 'data-scheme': mode,
          onclick: function () {
            state.scheme = mode === 'auto' ? '' : mode
            document.documentElement.style.removeProperty('color-scheme')
            commit('Scheme: ' + mode)
          },
        }),
      )
    })
    presetsPane.appendChild(el('div', { class: 'fbts-section' }, [
      el('div', { class: 'fbts-section-title', text: 'Appearance' }),
      schemeRow,
      el('div', { class: 'fbts-note', text: 'Presets set the full palette. Color scheme only tells the app how to render native controls and scrollbars.' }),
    ]))

    /* ---- Colors tab ---- */
    var colorsPane = pane('colors')
    colorsPane.appendChild(el('div', { class: 'fbts-note', text: 'Changes apply instantly and are saved automatically. \u21ba resets a single token.' }))
    var colorPanels = {}

    GROUP_ORDER.forEach(function (group) {
      var tokens = CURATED.filter(function (x) {
        return x.group === group
      })
      if (!tokens.length) return
      var isOpen = group === 'Surfaces' || group === 'Brand'
      var acc = el('div', { class: 'fbts-accordion' + (isOpen ? ' open' : '') })
      var headEl = el('div', { class: 'fbts-accordion-head' }, [
        el('span', { class: 'chev', text: '\u25b6' }),
        el('span', { text: group }),
        el('span', { class: 'grow', style: 'flex:1' }),
        el('span', { style: 'color:var(--fbts-faint);font-size:10px', text: String(tokens.length) }),
      ])
      headEl.addEventListener('click', function () { acc.classList.toggle('open') })
      var bodyEl = el('div', { class: 'fbts-accordion-body' })
      tokens.forEach(function (tk) {
        var row = tk.kind === 'color' ? colorRow(tk.name, tk.label) : textRow(tk.name, tk.label)
        bodyEl.appendChild(row)
      })
      acc.appendChild(headEl)
      acc.appendChild(bodyEl)
      colorsPane.appendChild(acc)
      colorPanels[group] = { acc: acc, body: bodyEl, tokens: tokens }
    })

    /* ---- Layout tab ---- */
    var layoutPane = pane('layout')
    var RADIUS_TOKENS = ['--radius-xs', '--radius-sm', '--radius-md', '--radius-lg', '--radius-xl', '--radius-control', '--radius-chrome-control', '--radius-popup', '--radius-composer', '--radius-dialog', '--radius-block', '--radius-inline']
    var FONT_TOKENS = ['--font-size-caption', '--font-size-label', '--font-size-ui', '--font-size-body', '--font-size-message', '--font-size-title', '--font-size-heading', '--font-size-display']
    var SIZE_TOKENS = ['--chat-content-max-width', '--explorer-width', '--catalog-width', '--tabbar-height', '--control-height-md', '--sidebar-row-gap']

    function layoutGroup(title, tokens) {
      var acc = el('div', { class: 'fbts-accordion open' })
      var headEl = el('div', { class: 'fbts-accordion-head' }, [el('span', { class: 'chev', text: '\u25b6' }), el('span', { text: title })])
      headEl.addEventListener('click', function () { acc.classList.toggle('open') })
      var bodyEl = el('div', { class: 'fbts-accordion-body' })
      tokens.forEach(function (name) {
        var meta = CURATED.filter(function (x) { return x.name === name })[0]
        bodyEl.appendChild(textRow(name, meta ? meta.label : name))
      })
      acc.appendChild(headEl)
      acc.appendChild(bodyEl)
      return acc
    }

    function radiusPreset(title, value) {
      return el('button', {
        class: 'fbts-btn', text: title,
        onclick: function () {
          ['--radius-xs', '--radius-sm', '--radius-md', '--radius-lg', '--radius-xl', '--radius-control', '--radius-chrome-control', '--radius-popup', '--radius-composer', '--radius-dialog', '--radius-block', '--radius-inline']
            .forEach(function (t) { state.layout[t] = value })
          commit('Radius: ' + title)
        },
      })
    }

    function fontScale(factor) {
      var base = { '--font-size-caption': 10, '--font-size-label': 11, '--font-size-ui': 12, '--font-size-body': 13, '--font-size-message': 14, '--font-size-title': 15, '--font-size-heading': 20, '--font-size-display': 24 }
      return el('button', {
        class: 'fbts-btn', text: Math.round(factor * 100) + '%',
        onclick: function () {
          for (var k in base) state.layout[k] = Math.round(base[k] * factor * 100) / 100 + 'px'
          commit('Text scale: ' + Math.round(factor * 100) + '%')
        },
      })
    }

    layoutPane.appendChild(el('div', { class: 'fbts-section' }, [
      el('div', { class: 'fbts-section-title', text: 'Corner radius' }),
      el('div', { class: 'fbts-actions', style: 'border:none;background:none;padding:0' }, [
        radiusPreset('Square', '0px'),
        radiusPreset('Subtle', '6px'),
        radiusPreset('Round', '12px'),
        radiusPreset('Pill', '24px'),
      ]),
    ]))
    layoutPane.appendChild(el('div', { class: 'fbts-section' }, [
      el('div', { class: 'fbts-section-title', text: 'Text scale' }),
      el('div', { class: 'fbts-actions', style: 'border:none;background:none;padding:0' }, [
        fontScale(0.9), fontScale(1), fontScale(1.1), fontScale(1.2),
      ]),
    ]))
    layoutPane.appendChild(el('div', { class: 'fbts-section' }, [
      el('div', { class: 'fbts-section-title', text: 'Typography' }),
      layoutGroup('Fonts', ['--font-sans', '--font-mono']),
      layoutGroup('Font sizes', FONT_TOKENS),
    ]))
    layoutPane.appendChild(el('div', { class: 'fbts-section' }, [
      el('div', { class: 'fbts-section-title', text: 'Radii' }),
      layoutGroup('Radius tokens', RADIUS_TOKENS),
    ]))
    layoutPane.appendChild(el('div', { class: 'fbts-section' }, [
      el('div', { class: 'fbts-section-title', text: 'Dimensions' }),
      layoutGroup('Sizing', SIZE_TOKENS),
    ]))

    /* ---- Advanced tab ---- */
    var advPane = pane('advanced')
    advPane.appendChild(el('div', { class: 'fbts-section-title', text: 'Raw CSS' }))
    var rawArea = el('textarea', { class: 'fbts-textarea', spellcheck: 'false', placeholder: '/* e.g. .composer { backdrop-filter: blur(20px); } */' })
    rawArea.value = state.raw || ''
    var rawTimer = null
    rawArea.addEventListener('input', function () {
      state.raw = rawArea.value
      clearTimeout(rawTimer)
      rawTimer = setTimeout(function () {
        applyState()
        saveState(state)
      }, 250)
    })
    advPane.appendChild(rawArea)
    advPane.appendChild(el('div', { class: 'fbts-note', text: 'Injected as a <style> tag. Anything CSS can do, this can do.' }))

    advPane.appendChild(el('div', { class: 'fbts-section', style: 'margin-top:16px' }, [
      el('div', { class: 'fbts-section-title', text: 'All tokens' }),
      el('div', { class: 'fbts-note', text: 'Every custom property Freebuff ships. Non-colour values are edited as text.' }),
    ]))
    var search = el('input', { class: 'fbts-search', type: 'text', placeholder: 'Filter tokens\u2026' })
    advPane.appendChild(search)
    var advList = el('div')
    advPane.appendChild(advList)
    var advRows = []
    ALL_TOKENS.forEach(function (name) {
      var meta = CURATED.filter(function (x) { return x.name === name })[0]
      var label = meta ? meta.label : name
      var row = meta && meta.kind === 'color' ? colorRow(name, label) : textRow(name, label)
      row.dataset.search = name.toLowerCase()
      advRows.push(row)
      advList.appendChild(row)
    })
    search.addEventListener('input', function () {
      var q = search.value.trim().toLowerCase()
      advRows.forEach(function (r) { r.style.display = !q || r.dataset.search.indexOf(q) !== -1 ? '' : 'none' })
    })

    /* ---- Export tab ---- */
    var ioPane = pane('io')
    var nameInput = el('input', { class: 'fbts-search', type: 'text', placeholder: 'Theme name' })
    nameInput.value = state.name || 'Custom'
    nameInput.addEventListener('input', function () {
      state.name = nameInput.value
      saveState(state)
    })
    ioPane.appendChild(el('div', { class: 'fbts-section-title', text: 'Theme name' }))
    ioPane.appendChild(nameInput)

    var exportArea = el('textarea', { class: 'fbts-textarea', spellcheck: 'false', readonly: 'readonly' })
    ioPane.appendChild(el('div', { class: 'fbts-section-title', style: 'margin-top:14px', text: 'Theme JSON' }))
    ioPane.appendChild(exportArea)
    ioPane.appendChild(el('div', { class: 'fbts-note', text: 'Paste a theme here and press Import to load it.' }))
    ioPane.appendChild(el('div', { class: 'fbts-actions', style: 'border:none;background:none;padding:10px 0 0' }, [
      el('button', {
        class: 'fbts-btn primary', text: 'Import',
        onclick: function () {
          try {
            var parsed = normalizeState(JSON.parse(exportArea.value))
            state = parsed
            applyState()
            saveState(state, true)
            rebuild()
            showToast('Theme imported')
          } catch (e) {
            showToast('Invalid JSON')
          }
        },
      }),
      el('button', {
        class: 'fbts-btn', text: 'Download .json',
        onclick: function () {
          download(JSON.stringify(state, null, 2), (state.name || 'theme') + '.freebuff-theme.json')
          showToast('Downloaded')
        },
      }),
      el('button', {
        class: 'fbts-btn', text: 'Copy CSS',
        onclick: function () {
          var css = cssText()
          if (navigator.clipboard) navigator.clipboard.writeText(css).then(function () { showToast('CSS copied') }).catch(function () { showToast('Copy failed') })
          else showToast('Clipboard unavailable')
        },
      }),
    ]))

    /* ---- footer ---- */
    var footer = el('div', { class: 'fbts-actions' }, [
      el('button', {
        class: 'fbts-btn danger', text: 'Reset all',
        onclick: function () {
          resetAll()
          rebuild()
          showToast('Reset to Freebuff defaults')
        },
      }),
      el('button', {
        class: 'fbts-btn', text: 'Check for updates',
        onclick: function () { checkForUpdates(true) },
      }),
      el('div', { class: 'grow' }),
      el('span', {
        class: 'fbts-note',
        style: 'margin:0',
        text: 'Unofficial community extension, not made by Freebuff.',
      }),
    ])

    panel.appendChild(head)
    panel.appendChild(el('div', { class: 'fbts-page-main' }, [tabsBar, body]))
    panel.appendChild(footer)

    document.addEventListener(
      'keydown',
      function (e) {
        if (e.ctrlKey && e.altKey && e.shiftKey && (e.key === 'F' || e.key === 'f')) {
          e.preventDefault()
          togglePage()
        }
        if (e.key === 'Escape') {
          // Esc backs out of the popup first, then closes the page.
          if (modalWrap.classList.contains('show')) {
            e.preventDefault()
            closePopup()
          } else if (panel.classList.contains('open')) {
            e.preventDefault()
            togglePage(false)
          }
        }
      },
      true,
    )

    /* ---- refresh helpers ---- */
    function presetSwatches(p) {
      if (!p.colors) return ['#0a0a0a', '#1c1c1c', '#b5cea5', '#ebebeb']
      return [p.colors['--bg'], p.colors['--surface'], p.colors['--brand'], p.colors['--text'], p.colors['--danger']]
        .filter(Boolean)
        .map(function (c) { return toCss(c) })
    }

    function refreshActive() {
      // preset cards
      Array.prototype.forEach.call(presetsGrid.children, function (card) {
        card.classList.toggle('active', card.dataset.preset === state.preset)
      })
      // scheme buttons
      Array.prototype.forEach.call(schemeRow.querySelectorAll('[data-scheme]'), function (btn) {
        var mode = btn.dataset.scheme
        var active = mode === 'auto' ? !state.scheme : state.scheme === mode
        btn.style.borderColor = active ? 'var(--fbts-accent)' : ''
      })
      // export area
      exportArea.value = JSON.stringify(state, null, 2)
      schemeLabel.textContent = 'v' + VERSION + '  \u00b7  ' + (state.preset === 'default' ? 'custom' : state.preset)
    }

    function rebuild() {
      // Simplest correct approach for a full-state swap: reload the pane values.
      Object.keys(tabPanes).forEach(function (k) {
        var p = tabPanes[k]
        if (k === 'colors' || k === 'advanced' || k === 'layout') {
          // re-seed every row from state
          Array.prototype.forEach.call(p.querySelectorAll('[data-token]'), function (row) {
            var token = row.dataset.token
            var colorInput = row.querySelector('input[type=color]')
            if (colorInput) {
              var c = currentColorFor(token)
              colorInput.value = c.hex
              var txt = row.querySelector('input[type=text]')
              if (txt) txt.value = c.hex
              var alpha = row.querySelector('input[type=range]')
              if (alpha) alpha.value = String(Math.round(c.a * 100))
            } else {
              var input = row.querySelector('input[type=text]')
              if (input) input.value = state.layout[token] || ''
            }
          })
        }
      })
      rawArea.value = state.raw || ''
      nameInput.value = state.name || ''
      refreshActive()
    }

    function download(text, filename) {
      var blob = new Blob([text], { type: 'application/json' })
      var url = URL.createObjectURL(blob)
      var a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      setTimeout(function () {
        URL.revokeObjectURL(url)
        a.remove()
      }, 1000)
    }

    function togglePage(force) {
      var open = typeof force === 'boolean' ? force : !panel.classList.contains('open')
      if (open) fitToWorkspace(panel)
      panel.classList.toggle('open', open)
      root.classList.toggle('fbts-page-open', open)
      if (railButton) {
        if (open) {
          // Only one destination should look active: take the highlight off the
          // app's own rail entries. React restores its own on the next
          // navigation, so this never fights the shell.
          var others = document.querySelectorAll('.shell-nav-button[aria-current]')
          for (var i = 0; i < others.length; i++) {
            if (others[i] !== railButton) others[i].removeAttribute('aria-current')
          }
          railButton.setAttribute('aria-current', 'page')
        } else {
          railButton.removeAttribute('aria-current')
        }
      }
      if (open) refreshActive()
    }

    ui = { toggle: togglePage, refresh: refreshActive, checkForUpdates: checkForUpdates }

    // The sidebar entry is the only entry point; watchRail() keeps re-attaching
    // it if React ever replaces the rail.
    installRailButton()

    // Keep the page glued to the workspace frame: on window resize and whenever
    // the shell decides to re-lay itself out (sidebar collapse, compact mode).
    pagePanel = panel
    fitToWorkspace(panel)
    window.addEventListener('resize', fitPage)
    try {
      var frameEl = document.querySelector('.workspace-frame') || document.querySelector('.settings-frame')
      new ResizeObserver(fitPage).observe(frameEl || document.documentElement)
    } catch (e) {}

    selectTab('presets')
    refreshActive()

    // One quiet check per session, well clear of the app's own startup work.
    setTimeout(function () {
      checkForUpdates(false)
    }, 2500)

    // Small surface for power users and for the sandbox tests.
    window.__FREEBUFF_THEME_STUDIO_API__ = {
      version: VERSION,
      checkForUpdates: checkForUpdates,
      state: function () {
        return clone(state)
      },
      /** Pretend the update feed reported this version. */
      simulateUpdate: function (v) {
        remoteVersion = cleanVersion(v)
        if (remoteVersion) showUpdatePopup()
      },
      /** Same, but a skipped version: it only shows the corner button. */
      simulateChip: function (v) {
        remoteVersion = cleanVersion(v)
        showChip(remoteVersion)
      },
    }
  }

  /* ------------------------------------------------------------------ *
   * Boot
   * ------------------------------------------------------------------ */

  function boot() {
    if (!document.body) return
    try {
      mount()
      closePageOnExternalNav()
      watchRail()
    } catch (err) {
      // Never let a UI failure take the host app down with it.
      console.error('[freebuff-theme-studio] failed to mount', err)
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot)
  } else {
    boot()
  }
})();
