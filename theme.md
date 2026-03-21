# Theme Reference

This document describes the design system and visual theme used in Yantr.

---

## Color Tokens

CSS custom properties defined in `src/assets/main.css`.

### Light Mode (`:root`)

| Token            | Value                        | Usage                     |
|------------------|------------------------------|---------------------------|
| `--bg-body`      | `#ffffff`                    | Page background           |
| `--bg-glass`     | `rgba(255,255,255,0.78)`     | Frosted glass surfaces    |
| `--border-glass` | `rgba(15,23,42,0.08)`        | Glass element borders     |
| `--text-primary` | `#0f172a`                    | Headings and body text    |
| `--text-secondary`| `#475569`                   | Labels, captions, hints   |
| `--surface`      | `#ffffff`                    | Cards, modals             |
| `--surface-muted`| `#f1f5f9`                    | Subtle backgrounds        |

### Dark Mode (`:root.dark`)

| Token            | Value                        | Usage                     |
|------------------|------------------------------|---------------------------|
| `--bg-body`      | `#0b1220`                    | Page background           |
| `--bg-glass`     | `rgba(15,23,42,0.6)`         | Frosted glass surfaces    |
| `--border-glass` | `rgba(148,163,184,0.2)`      | Glass element borders     |
| `--text-primary` | `#e2e8f0`                    | Headings and body text    |
| `--text-secondary`| `#94a3b8`                   | Labels, captions, hints   |
| `--surface`      | `#0f172a`                    | Cards, modals             |
| `--surface-muted`| `#111827`                    | Subtle backgrounds        |

Dark mode is activated by adding the `dark` class to `:root` (configured via `darkMode: 'class'` in Tailwind).

---

## Banned Patterns

The following are **absolutely forbidden** in all components, views, and utilities. Remove any existing usage immediately when encountered.

| Rule | What is banned |
|------|----------------|
| **No gradients** | `background: linear-gradient(…)`, `background: radial-gradient(…)`, `background: conic-gradient(…)`, `bg-gradient-*` Tailwind classes, `from-*`/`via-*`/`to-*` Tailwind utilities, `border-image` gradients — all banned without exception. Use flat `--surface`, `--surface-muted`, or a single solid color instead. |
| **No glass morphism** | See [Glass Morphism](#glass-morphism) section — `.glass`, `.glass-dark`, `backdrop-filter`, `backdrop-blur` are banned. |
| **No borders on sections or cards** | Do not add `border`, `border-*`, `ring`, or `ring-*` to section wrappers or card containers unless a border is explicitly required by the design spec for that component. Use `.smooth-shadow` / `.smooth-shadow-lg` for elevation instead. |

> **Rule:** Never use a gradient for any purpose — backgrounds, borders, text, overlays, or decorative elements. Flat color is always the correct choice.

---

## Focus / Accent Color

| State      | Color               | Ring shadow                          |
|------------|---------------------|--------------------------------------|
| Light focus| `#3b82f6` (blue-500)| `0 0 0 4px rgba(59,130,246,0.20)`    |
| Dark focus | `#60a5fa` (blue-400)| `0 0 0 4px rgba(59,130,246,0.35)`    |

Applied to `input` and `textarea` on `:focus`.

---

## Typography

**Font stack** (applied globally via `*`):

```
-apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', sans-serif
```

- Anti-aliasing: `-webkit-font-smoothing: antialiased` + `-moz-osx-font-smoothing: grayscale`
- System-native rendering — no custom web fonts loaded.

---

## Motion & Easing

| Token               | Value                          | Description                  |
|---------------------|--------------------------------|------------------------------|
| `--yantra-ease`     | `cubic-bezier(0.16,1,0.3,1)`  | Spring-like default easing   |
| `--yantra-dur-fast` | `200ms`                        | Micro-interactions (buttons) |
| `--yantra-dur`      | `280ms`                        | Standard transitions         |
| `--yantra-dur-enter`| `300ms`                        | Entrance animations          |

All `button` and `a` elements use `--yantra-dur-fast` by default.

### Named Animations

| Name          | Duration  | Description                             |
|---------------|-----------|-----------------------------------------|
| `fadeIn`      | `400ms`   | Fade + slide up 6 px (Tailwind keyframe)|
| `animate-fadeIn`| `300ms` | Fade + slide up 12 px (CSS class)       |
| `progress`    | `1.5s`    | Horizontal scale 0→1 (progress bars)    |
| `ripple-ping` | `650ms`   | Scale pulse with fade (ripple effect)   |
| `spin-slow`   | `3s`      | Slow infinite rotation                  |
| `status-dot`  | `2s`      | Opacity pulse for live status indicators|

---

## Glass Morphism

> **Deprecated — do not use.** Glass morphism (`.glass`, `.glass-dark`, `backdrop-filter`, `backdrop-blur`) is banned from the design system. Remove any existing usage when encountered.

Use solid `--surface` / `--surface-muted` backgrounds with `.smooth-shadow` or `.smooth-shadow-lg` instead.

---

## Hover Animations

Every interactive component **must** include a hover animation. No component should be visually static on hover.

### Required hover behaviors

| Component type        | Minimum hover effect                                      |
|-----------------------|-----------------------------------------------------------|
| Buttons               | `scale(1.03)` + lightened/darkened background             |
| Cards                 | `scale(1.02)` + `.smooth-shadow-lg` (use `.card-hover`)   |
| List / grid items     | `scale(1.01)` + subtle background tint                    |
| Icon buttons / links  | `scale(1.1)` + opacity or color shift                     |
| Navigation items      | Background fill + color shift                             |
| Input wrappers        | Border color shift to `--text-secondary`                  |

### Implementation

All hover transitions must use the existing motion tokens:

```css
transition: transform var(--yantra-dur-fast) var(--yantra-ease),
            box-shadow var(--yantra-dur-fast) var(--yantra-ease),
            background-color var(--yantra-dur-fast) var(--yantra-ease);
```

The `.card-hover` utility class already covers the standard card pattern. For other components, apply the transition inline or via a scoped CSS rule.

> **Rule:** If you add or edit a component, verify it has a visible hover state before committing.

---

## Elevation / Shadows

| Class              | Value                                  |
|--------------------|----------------------------------------|
| `.smooth-shadow`   | `0 1px 3px rgba(0,0,0,0.06)`          |
| `.smooth-shadow-lg`| `0 4px 16px rgba(0,0,0,0.10)`         |
| `.card-hover`      | On hover: `scale(1.02)` + `0 8px 24px rgba(0,0,0,0.12)` |

---

## Scrollbar

Custom styled (WebKit) — 8 px, rounded track and thumb.

| State        | Light thumb                    | Dark thumb                     |
|--------------|--------------------------------|--------------------------------|
| Default      | `rgba(15,23,42,0.25)`          | `rgba(148,163,184,0.40)`       |
| Hover        | `rgba(15,23,42,0.35)`          | `rgba(148,163,184,0.55)`       |

---

## Icons

Lucide icons are the **only** icon library used. Icons must be added wherever they aid scannability or clarify intent — they are not optional decoration.

### When to use an icon (required)

| Context | Requirement |
|---------|-------------|
| Buttons & CTAs | Every button must have a leading icon unless width is critically constrained |
| Navigation items | Each nav link must have an icon |
| Section headings | Use an icon to the left of every major section/page title |
| Status & feedback | Use icons for success, error, warning, info states (never text-only) |
| Empty states | Include a large (24–32 px) icon above the empty-state message |
| Form labels | Use an icon alongside labels for common fields (search, email, date, etc.) |
| List items with actions | Delete, edit, copy, share actions must use icon buttons |
| Toast / alert messages | Leading icon required to communicate severity |

### Sizing conventions

| Use case          | Size  |
|-------------------|-------|
| Inline / button   | 16 px |
| Standard UI       | 20 px |
| Section headings  | 20–24 px |
| Empty states      | 32 px |

### Global style override

```css
svg.lucide {
  stroke-width: 1.75;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}
```

> **Rule:** If you add or edit a component, confirm every action, label, and status has a matching Lucide icon before committing.

---

## Mobile UI

Every view and component **must look and work great on mobile**. Mobile is a first-class target, not an afterthought.

### Layout rules

| Rule | Requirement |
|------|-------------|
| **Responsive breakpoints** | Design mobile-first — base styles target small screens, `md:`/`lg:` Tailwind prefixes add desktop enhancements |
| **Minimum tap target** | All interactive elements (buttons, links, icons) must be at least **44 × 44 px** |
| **No horizontal scroll** | No component may cause the page to scroll horizontally on any screen width |
| **Readable text** | Minimum `14px` font size for body copy; never below `12px` for any visible text |
| **Full-width inputs** | Form inputs and selects must be `w-full` on mobile |
| **Stack on small screens** | Multi-column grids must collapse to a single column on `sm` and below |

### Touch & spacing

- Apply `.touch-manipulation` to all buttons and interactive elements to disable double-tap zoom.
- Apply `.mobile-scroll` to any scrollable overflow container.
- Apply `.safe-area-inset-bottom` to fixed/sticky bottom bars to respect iOS home indicator.
- Use `p-4` minimum padding inside cards and list items so content never touches the screen edge.

### Testing requirement

> **Rule:** Before committing any view or component change, resize the browser to 375 px width (iPhone SE) and verify the layout is usable, readable, and unbroken.

---

## Utility Classes

| Class                 | Purpose                                    |
|-----------------------|--------------------------------------------|
| `.glass`              | ~~Frosted glass surface~~ **Banned — do not use** |
| `.card-hover`         | Scale + shadow on hover                    |
| `.smooth-shadow`      | Subtle 1 px drop shadow                    |
| `.smooth-shadow-lg`   | Larger 4 px drop shadow                    |
| `.animate-fadeIn`     | Entrance fade + slide animation            |
| `.status-dot`         | Pulsing dot for live status                |
| `.line-clamp-1/2`     | Single or double line text truncation      |
| `.safe-area-inset-bottom` | iOS safe-area bottom padding           |
| `.touch-manipulation` | Disables double-tap zoom for touch targets |
| `.mobile-scroll`      | Momentum scrolling on iOS                  |
