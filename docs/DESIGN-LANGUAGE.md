# Lyn Design Language

A brand brief for reproducing Lyn's UI in any tool — including Claude Design (claude.ai/design). The Lyn components are Vue, which Claude Design cannot import, so this document captures the *design language* itself: tokens, type, shape, motion, and component patterns, in a form a React-building agent can apply to match the brand.

The source of truth is `frontend/src/style.css` (tokens, fonts), `frontend/src/themes.ts` (palettes), and `frontend/src/components/ui/*.vue` (component idioms). When those change, update this file.

## Foundations

Dark-first, Material-3-flavored. Every color is a CSS custom property in two layers:

- **Base palette** (`--lyn-*`) — set at runtime from the active theme (`themes.ts`).
- **Semantic M3 tokens** (`--m3-*`) — derived from the base palette via `color-mix`. Components only ever reference `--m3-*`, never raw hex.

Style with these tokens through `var(--m3-*)` (or Tailwind's arbitrary-value syntax `bg-(--m3-surface-container)`). Never hard-code hex in components — read the token.

### Semantic tokens (the component-facing vocabulary)

| Token | Role | Derivation (from base palette) |
|---|---|---|
| `--m3-surface` | window background | `--lyn-bg` |
| `--m3-surface-container` | card / panel fill | `mix(bg 95%, text)` |
| `--m3-surface-container-high` | raised fill, inputs | `mix(bg 90%, text)` |
| `--m3-surface-container-highest` | tracks, badges | `mix(bg 85%, text)` |
| `--m3-on-surface` | primary text | `--lyn-text` |
| `--m3-on-surface-variant` | secondary text, icons | `mix(text 62%, transparent)` |
| `--m3-outline` | hairline dividers | `mix(text 12%, transparent)` |
| `--m3-outline-strong` | input / control borders | `mix(text 24%, transparent)` |
| `--m3-accent` | primary action, focus, active | `--lyn-accent` |
| `--m3-accent-container` | selected-card fill | `mix(accent 24%, bg)` |
| `--m3-on-accent` | text/icon on accent | `--lyn-bg` |
| `--m3-state` | hover wash | `mix(text 8%, transparent)` |
| `--m3-state-strong` | stronger hover wash | `mix(text 12%, transparent)` |
| `--m3-error` | destructive / error | `#ff897d` |

Key relationships to preserve: hover = translucent text wash (`--m3-state`), never a solid color. Selection = accent at low opacity (`--m3-accent-container`). Focus = 1px accent ring (`box-shadow: 0 0 0 1px var(--m3-accent)`) plus an accent border. On-accent text is the *background* color, not white — accents are bright and light-on-dark.

### Base palette — themes

Eight built-in themes; **Power Run is the default**. Each defines 8 colors that feed the M3 layer above.

| Theme | bg | panel | panelAlt | border | text | muted | accent | selected |
|---|---|---|---|---|---|---|---|---|
| **Power Run** | `#202020` | `#1f1f1f` | `#2d2d2d` | `#515151` | `#ffffff` | `#ffffff99` | `#60cdff` | `#333333` |
| Tron Legacy | `#071016` | `#0b151c` | `#14242e` | `#20d8ff66` | `#d7f7ff` | `#8ab8c8` | `#ffb20d` | `#d7f7ff14` |
| Dracula | `#282a36` | `#21222c` | `#343746` | `#44475a` | `#f8f8f2` | `#6272a4` | `#bd93f9` | `#44475a` |
| Nord | `#2e3440` | `#292e39` | `#3b4252` | `#434c5e` | `#eceff4` | `#7b88a1` | `#88c0d0` | `#434c5e` |
| Catppuccin Mocha | `#1e1e2e` | `#181825` | `#313244` | `#45475a` | `#cdd6f4` | `#a6adc8` | `#cba6f7` | `#313244` |
| Gruvbox Dark | `#282828` | `#1d2021` | `#3c3836` | `#504945` | `#ebdbb2` | `#928374` | `#fabd2f` | `#3c3836` |
| Tokyo Night | `#1a1b26` | `#16161e` | `#292e42` | `#3b4261` | `#c0caf5` | `#565f89` | `#7aa2f7` | `#292e42` |
| One Dark | `#282c34` | `#21252b` | `#2c313a` | `#3e4451` | `#abb2bf` | `#5c6370` | `#61afef` | `#3e4451` |

## Typography

Two families, both shipped as `woff2` (`frontend/src/fonts/`):

- **Sans** — `"Roboto Flex", "Roboto", ui-sans-serif, system-ui, sans-serif` (variable weight 100–1000). The UI font.
- **Mono** — `"Roboto Mono", ui-monospace, monospace`. File paths and code only.

Antialiased, `font-synthesis: none`, `text-rendering: optimizeLegibility`. The type scale is tight and tracking-positive — these exact steps recur across the kit:

| Role | Size / line-height | Tracking | Weight | Color |
|---|---|---|---|---|
| Section header | 11px / 16px, UPPERCASE | `0.55px` | bold | `--m3-accent` |
| Caption / field label | 12px / 16px | `0.4px` | normal | `--m3-on-surface-variant` |
| Body / control text | 13px / 18px | `0.1px` | medium on actions | `--m3-on-surface` |
| Badge / micro | 10px / 14px | — | normal | `--m3-on-surface-variant` |
| Path (mono) | 12px / 16px | — | normal | `--m3-on-surface` |

13px/18px with `0.1px` tracking is the workhorse body size — default to it for buttons, rows, and inputs.

## Shape, spacing, motion

- **Radius**: pills/buttons/toggles `rounded-full` (999px); cards `14px`; inputs/selects `8px` (`rounded-lg`).
- **Control heights**: primary buttons 34px; secondary/icon buttons 32px; selects 36px; rows min 44px (touch target); toggles 20×36px.
- **Spacing**: rows pad `14px` horizontal, `10px` vertical; cards group with `1.5` (6px) gaps; section content stacks at `gap-1.5`.
- **Dividers**: hairline pseudo-element (`after:h-px after:bg-(--m3-outline)`) inset `14px`, hidden on `last:` — not a `border-bottom`.
- **Motion**: `140ms ease` is the standard transition (some 160ms on toggles). Only animate what changes — `transition-[background]`, `transition-[background,border-color]`, never `transition-all`.

## Styling idiom

This is a **Tailwind v4 utility-class** system layered over the M3 token CSS-variables. There is no separate class library — components compose Tailwind utilities, reaching tokens via the arbitrary-value form `bg-(--m3-…)`, `text-(--m3-…)`, `border-(--m3-…)`. Read `frontend/src/style.css` for the full token + `@theme` definitions before styling.

To match the brand in a React/Tailwind build:

1. Include the M3 token block (below) at `:root`, plus the two `@font-face` rules.
2. Build controls from Tailwind utilities that reference those tokens — never literal hex.
3. Follow the patterns in the catalog below for the recurring shapes.

## Component catalog

The 14 primitives in `frontend/src/components/ui/`. APIs are the real `defineProps`; reproduce the *behavior and look*, not the Vue mechanics. `slot` ≈ `children`; `v-model` ≈ a controlled value+onChange pair; `emit` ≈ a callback prop.

| Component | API | Pattern |
|---|---|---|
| `UiButton` | `variant: "text" \| "filled" \| "link" \| "add"`, `active`, `disabled` | Pill, 34px. `filled` = accent bg + on-accent text; `text` = transparent, accent text, `--m3-state` hover; `link` = outlined toggle, accent-filled when `active`; `add` = outlined with a `+`. `disabled:opacity-45`. |
| `UiIconButton` | `variant: "default" \| "danger"` | 32px circular, transparent. Icon tinted `--m3-on-surface-variant`; hover wash + tint to `--m3-on-surface` (default) or `--m3-error` (danger). SVG sized 18–20px. |
| `UiCard` | children | Container: `rounded-[14px]`, `bg-(--m3-surface-container)`, `overflow-hidden`, column flex. Wraps rows. |
| `UiCardRow` | `as: "div" \| "label"` | A row inside a card with the inset hairline divider; `14px/10px` padding. |
| `UiSection` | `title`, `actions` slot | Labeled group: uppercase accent 11px header + optional right-aligned actions, then content at `gap-1.5`. |
| `UiField` | `label`, `inline`, `labelClass` | Labeled control wrapper with divider. `inline` → label left, control right (`grid-cols-[1fr_auto]`); else stacked. Label 13px on-surface, caption tracking. |
| `UiSelect` | `v-model` + options children | 36px native `<select>`, `rounded-lg`, strong outline, container-high fill, custom chevron. Focus = accent border + 1px ring. |
| `UiSlider` | `min`, `max`, `step` + `v-model.number` | Range input. 4px track (`--m3-surface-container-highest`), 14px accent thumb ringed in container color. |
| `UiToggle` | `disabled` + `v-model` | 20×36 switch. Outlined track → fills `--m3-accent`, thumb slides, when on. 160ms. |
| `UiToggleRow` | `label`, `disabled` + `v-model` | Full-width label-left / toggle-right row, 13px body. |
| `UiNavRow` | `icon`, `label`, `as`, `active` | 44px nav item: leading icon + label, hover wash, divider, accent text when `active`. |
| `UiPathRow` | `path`, `badge`, `@remove` | Mono path with folder icon, optional pill badge, trailing danger icon-button to remove. Truncates with ellipsis. |
| `UiOptionCard` | `icon`, `title`, `description`, `selected`, `disabled`, `tone`, `@select` | Selectable 14px-radius card. Outlined → accent-container fill + accent icon when `selected`; `tone:"error"` tints icon `--m3-error`. |
| `UiSwatch` | `accent`, `bg`, `active`, `label` | 22px round theme chip filled with its accent; `active` adds a double accent ring + check. |

### Build snippet (React + Tailwind)

A primary action and a selectable card in the Lyn idiom:

```tsx
<button
  type="button"
  className="inline-flex h-[34px] items-center justify-center gap-2 rounded-full
             bg-(--m3-accent) px-5 text-[13px]/[18px] font-medium tracking-[0.1px]
             text-(--m3-on-accent) transition-[background] duration-140 ease-[ease]
             disabled:opacity-45 [&_svg]:size-4 [&_svg]:fill-current">
  Run
</button>

<button
  type="button"
  onClick={onSelect}
  className={`relative grid min-h-23 gap-1 rounded-[14px] border py-3 pr-10 pl-3.5
              text-left text-(--m3-on-surface) transition-[background,border-color]
              duration-140 ease-[ease] ${selected
                ? "border-transparent bg-(--m3-accent-container)"
                : "border-(--m3-outline) bg-(--m3-surface-container) hover:bg-(--m3-surface-container-high)"}`}>
  <span className="text-[13px]/[18px] tracking-[0.1px]">Project folder</span>
  <span className="text-[12px]/4 tracking-[0.4px] text-(--m3-on-surface-variant)">
    Watch this directory for launchers
  </span>
</button>
```

### Token block (paste at `:root`)

Defaults to Power Run; swap the eight `--lyn-*` values for any theme above.

```css
@font-face {
  font-family: "Roboto Flex"; font-style: normal; font-weight: 100 1000;
  font-display: swap; src: url("./fonts/roboto-flex-latin.woff2") format("woff2");
}
@font-face {
  font-family: "Roboto Mono"; font-style: normal; font-weight: 400;
  font-display: swap; src: url("./fonts/roboto-mono-latin.woff2") format("woff2");
}

:root {
  /* Base palette — Power Run (swap per theme) */
  --lyn-bg: #202020;
  --lyn-text: #ffffff;
  --lyn-accent: #60cdff;

  /* Semantic M3 tokens (components reference only these) */
  --m3-surface: var(--lyn-bg);
  --m3-surface-container: color-mix(in srgb, var(--lyn-bg) 95%, var(--lyn-text));
  --m3-surface-container-high: color-mix(in srgb, var(--lyn-bg) 90%, var(--lyn-text));
  --m3-surface-container-highest: color-mix(in srgb, var(--lyn-bg) 85%, var(--lyn-text));
  --m3-on-surface: var(--lyn-text);
  --m3-on-surface-variant: color-mix(in srgb, var(--lyn-text) 62%, transparent);
  --m3-outline: color-mix(in srgb, var(--lyn-text) 12%, transparent);
  --m3-outline-strong: color-mix(in srgb, var(--lyn-text) 24%, transparent);
  --m3-accent: var(--lyn-accent);
  --m3-accent-container: color-mix(in srgb, var(--lyn-accent) 24%, var(--lyn-bg));
  --m3-on-accent: var(--lyn-bg);
  --m3-state: color-mix(in srgb, var(--lyn-text) 8%, transparent);
  --m3-state-strong: color-mix(in srgb, var(--lyn-text) 12%, transparent);
  --m3-error: #ff897d;

  font-family: "Roboto Flex", "Roboto", ui-sans-serif, system-ui, sans-serif;
}
```
