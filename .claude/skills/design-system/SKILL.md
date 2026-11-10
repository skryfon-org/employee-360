---
name: design-system
description: >
  Employee360 visual design spec: flat solid color tokens (zero gradients), status/category
  colors, and base button primitives, expressed as Tailwind classes. Use whenever building or
  styling UI in `clients/*` or `packages/ui` — pages, components, forms, tables, badges, buttons.
---

# Employee360 Design System

Apply these tokens with Tailwind classes. **Plain solid colors only — no gradients, no
shadows-as-decoration.** Don't invent new colors; use the tokens below.

## 1. Color tokens

### Surfaces & canvas
| Token | Hex | Class | Use |
|---|---|---|---|
| App canvas | `#F8FAFC` | `bg-slate-50` | Flat page background |
| Surface default | `#FFFFFF` | `bg-white` | Cards, modals, panels |
| Surface subtle | `#F1F5F9` | `bg-slate-100` | Table heads, input fills, tab track |
| Surface selected | `#E2E8F0` | `bg-slate-200` | Selected rows, active nav pills |

### Borders
| Token | Hex | Class | Use |
|---|---|---|---|
| Default | `#E2E8F0` | `border-slate-200` | Layout grids, card frames |
| Strong | `#CBD5E1` | `border-slate-300` | Inputs, calendar cell lines, modal containers |
| Focus / keyline | `#0F172A` | `border-slate-900` | Focused inputs, active tab keyline |

### Text
| Token | Hex | Class | Use |
|---|---|---|---|
| High contrast | `#0F172A` | `text-slate-900` | Headings, dates, primary copy |
| Muted | `#475569` | `text-slate-600` | Helper text, column titles, empty states |
| Inverted | `#FFFFFF` | `text-white` | Text on solid dark buttons/chips |

### Category & status (badge = fill + border + text; dot uses the `-700` bg)
| Meaning | Fill | Border | Text / Dot |
|---|---|---|---|
| Mandatory / Gazetted | `bg-green-50` `#F0FDF4` | `border-green-300` `#86EFAC` | `text-green-700` / `bg-green-700` `#15803D` |
| Floating / Optional | `bg-amber-50` `#FFFBEB` | `border-amber-300` `#FDE68A` | `text-amber-700` / `bg-amber-700` `#B45309` |
| Company Offsite / Special | `bg-blue-50` `#EFF6FF` | `border-blue-300` `#BFDBFE` | `text-blue-700` / `bg-blue-700` `#1D4ED8` |
| Destructive / Error | `bg-red-50` `#FEF2F2` (action fill `bg-red-600` `#DC2626`) | `border-red-200` | `text-red-700` `#B91C1C` |

## 2. Foundations

- **Radius:** `rounded-sm` everywhere (buttons, inputs, badges, cards, modals). One radius.
- **Spacing scale:** 4 / 8 / 12 / 16 / 24 / 32 px (`1, 2, 3, 4, 6, 8`). No other values.
- **Type scale:** page title `text-xl font-semibold`, section heading `text-base font-semibold`,
  body/buttons/inputs `text-sm`, helper text/badges/table column titles `text-xs`.
- **Focus (all interactive elements):** `focus:outline-none focus-visible:ring-2
  focus-visible:ring-slate-900 focus-visible:ring-offset-2`. Destructive elements use
  `focus-visible:ring-red-600`. Never remove the outline without a ring.
- **Disabled:** `disabled:bg-slate-200 disabled:text-slate-500 disabled:border-slate-200
  disabled:cursor-not-allowed` (no hover/active change).
- **Links:** `text-slate-900 hover:text-slate-700` (no underline). Blue is reserved for the
  Company Offsite category — never use it for links or info alerts.
- **Dividers inside selected areas** (`bg-slate-200`): use `border-slate-300`, since
  `border-slate-200` would vanish.
- **Muted text** never lighter than `text-slate-600` (AA contrast on `slate-100`).

## 3. Primitives

### Buttons (height `h-9`, `rounded-sm`, `text-sm`)
```html
<!-- Primary -->
<button class="h-9 px-4 rounded-sm bg-slate-900 text-white text-sm font-semibold hover:bg-slate-800 active:bg-slate-950 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:bg-slate-200 disabled:text-slate-500 disabled:cursor-not-allowed">
  Button Label
</button>

<!-- Secondary / Outline -->
<button class="h-9 px-4 rounded-sm bg-white border border-slate-300 text-slate-800 text-sm font-medium hover:bg-slate-100 active:bg-slate-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900 focus-visible:ring-offset-2 disabled:bg-slate-100 disabled:text-slate-500 disabled:cursor-not-allowed">
  Button Label
</button>

<!-- Danger (outline) — use for the trigger of a destructive action -->
<button class="h-9 px-4 rounded-sm bg-white border border-red-300 text-red-600 text-sm font-medium hover:bg-red-50 active:bg-red-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 disabled:text-slate-500 disabled:border-slate-200 disabled:cursor-not-allowed">
  Delete
</button>

<!-- Danger (solid) — use for the final confirm button inside a confirmation modal -->
<button class="h-9 px-4 rounded-sm bg-red-600 text-white text-sm font-semibold hover:bg-red-700 active:bg-red-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-600 focus-visible:ring-offset-2 disabled:bg-slate-200 disabled:text-slate-500 disabled:cursor-not-allowed">
  Delete permanently
</button>
```
**Loading:** keep the label, add a leading spinner (`h-4 w-4 animate-spin`, `currentColor`),
set `disabled` and `aria-busy="true"`. Don't change the button width.

### Text input
```html
<input class="h-9 w-full px-3 rounded-sm bg-white border border-slate-300 text-sm text-slate-900 placeholder:text-slate-500 focus:outline-none focus:border-slate-900 focus:ring-1 focus:ring-slate-900 disabled:bg-slate-100 disabled:text-slate-500 disabled:cursor-not-allowed aria-[invalid=true]:border-red-600 aria-[invalid=true]:focus:ring-red-600" />
```
OTP: one input per digit, `h-10 w-10 text-center`, same border/focus/error styles.

### Form field (label + input + helper/error)
```html
<div class="flex flex-col gap-1">
  <label class="text-xs font-medium text-slate-900">Email</label>
  <!-- input -->
  <p class="text-xs text-slate-600">Helper text</p>
  <p class="text-xs text-red-700" role="alert">Error message</p> <!-- replaces helper on error -->
</div>
```
Bind `aria-invalid` and `aria-describedby` on the input to the message.

### Badge (category / status)
```html
<span class="inline-flex items-center gap-1.5 h-6 px-2 rounded-sm border text-xs font-medium bg-green-50 border-green-300 text-green-700">
  <span class="h-1.5 w-1.5 rounded-full bg-green-700"></span>Mandatory
</span>
```
Swap the color trio per the category table above.

### Inline alert / banner
```html
<div role="alert" class="flex gap-2 p-3 rounded-sm border bg-red-50 border-red-200 text-red-700 text-sm">
  <!-- icon --> <p>Message</p>
</div>
```
Always include an icon and text so color is never the only signal. For non-error alerts use
the neutral surface (`bg-slate-100 border-slate-300 text-slate-900`), not blue.

### Card, table, modal
- **Card:** `bg-white border border-slate-200 rounded-sm p-4` (or `p-6`).
- **Table:** wrapper is a card; `thead` `bg-slate-100 text-xs font-medium text-slate-600`;
  rows `border-t border-slate-200 text-sm text-slate-900`; hover `hover:bg-slate-50`;
  selected `bg-slate-200`.
- **Modal:** panel `bg-white border border-slate-300 rounded-sm p-6`, backdrop
  `bg-slate-900/50` (flat, not a gradient); title `text-base font-semibold`; actions
  right-aligned, secondary then primary, `gap-2`.

## 4. Theme aliases & shared layout (packages/ui)

Aliases live in `packages/ui/src/theme.css` (`@theme inline`, imported by both clients after
`@import "tailwindcss"`). Use them in new shared components; raw `slate-*` stays valid.

| Alias | Maps to | Utility examples |
|---|---|---|
| `canvas` | slate-50 | `bg-canvas` |
| `surface` / `surface-subtle` / `surface-selected` | white / slate-100 / slate-200 | `bg-surface` |
| `line` / `line-strong` | slate-200 / slate-300 | `border-line` |
| `ink` / `ink-muted` | slate-900 / slate-600 | `text-ink` |
| `accent` / `accent-hover` | slate-900 / slate-800 | `bg-accent` |

Shared layout components (import from `@employee360/ui`), use them instead of bespoke markup:
- `BrandMark` - logo tile + product name/subtitle (sidebar, headers, auth pages).
- `PageHeader` - page title, optional description, right-aligned `actions` (stacks on mobile).
- `Card` - `rounded-sm border border-line bg-surface`, padded `p-4 sm:p-6`.
- `UserBadge` - initial chip + email (email hidden below `sm`).
- `AuthLayout` - centered brand + card for all auth pages.

Layout conventions:
- Header `h-14`, sticky (`sticky top-0 z-20`), `border-b border-slate-200 bg-white`.
- Page gutters `px-4 sm:px-6 lg:px-8`, content `max-w-6xl`, vertical gap `gap-6`.
- Active nav item: `border-l-2 border-slate-900 bg-slate-200 font-semibold`; inactive items use
  `border-transparent` to avoid layout shift.
- No shadows and no `rounded-lg/md`; muted text never lighter than `slate-600`.

## Rules
- Wrap these in shared React components (`packages/ui` or the feature's `components/`) rather
  than repeating class strings; props select the variant.
- Color is never the only signal — pair it with text, and an icon or dot.
- Prefer theme aliases over raw `slate-*`/`green-*` classes once defined in the Tailwind
  config (e.g. `surface`, `border-strong`, `status-mandatory`), so palette changes and
  per-tenant theming stay a one-file change. Core aliases now exist (section 4); for status/category
  colors without aliases yet, use the raw classes above and don't invent new colors.
- Propose additions to this skill rather than diverging silently.
