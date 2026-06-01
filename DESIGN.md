# Design

## Design Principles

- **Restrained palette.** Tinted neutrals plus one deliberate accent. Nothing decorative.
- **Functional typography.** System font stack, tight scale ratio (1.125), hierarchy through weight and size.
- **State-complete components.** Every interactive element has default, hover, focus, active, disabled, loading, error states.
- **Respect motion budget.** 150–250ms ease-out transitions. No animation for animation's sake.

## Color

### Palette

| Role | Token | Hex (approx OKLCH) | Usage |
|---|---|---|---|
| Primary | `--color-primary` | `#1A56DB` (OKLCH 42% 0.17 265) | Primary actions, links, focus rings |
| Primary hover | `--color-primary-hover` | `#1648B8` | Button hover |
| Success | `--color-success` | `#059669` | Success states |
| Warning | `--color-warning` | `#D97706` | Warning states |
| Danger | `--color-danger` | `#DC2626` | Error, destructive actions |
| Background | `--color-background` | `#FFFFFF` | Main surface |
| Background subtle | `--color-background-subtle` | `#F9FAFB` | Alternate surface, sidebar tint |
| Border | `--color-border` | `#E5E7EB` | Dividers, input borders |
| Text primary | `--color-text-primary` | `#111827` | Headings, body |
| Text secondary | `--color-text-secondary` | `#6B7280` | Labels, captions |
| Text muted | `--color-text-muted` | `#9CA3AF` | Placeholders, disabled |

**Tint rule:** Neutrals are slightly cool (blue-tinted). Never pure gray. Text-primary is not `#000`.

### State Colors

- Hover: darken primary by ~8% for interactive elements
- Focus: 2px solid primary with 2px offset (WCAG 2.1 AA)
- Error: `--color-danger` with light red background tint `color-mix(in oklch, var(--color-danger) 10%, white)`
- Success: `--color-success` with light green background tint

## Typography

**Stack:** `-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif`

**Scale (1.125 ratio):**

| Step | Token | Size | Usage |
|---|---|---|---|
| xs | `--text-xs` | 0.75rem / 12px | Timestamps, badges |
| sm | `--text-sm` | 0.875rem / 14px | Labels, secondary UI |
| base | `--text-base` | 1rem / 16px | Body text |
| lg | `--text-lg` | 1.125rem / 18px | Subheadings |
| xl | `--text-xl` | 1.25rem / 20px | Section headings |
| 2xl | `--text-2xl` | 1.5rem / 24px | Page titles |
| 3xl | `--text-3xl` | 1.875rem / 30px | Hero headings |

**Weight usage:**
- Regular (400): body text
- Medium (500): labels, navigation
- Semibold (600): buttons, headings, emphasis

**Line length:** 65–75ch for prose. UI components can be denser.

## Spacing

Base unit: 4px. Scale: 1, 2, 3, 4, 6, 8, 12, 16, 24, 32.

| Token | Value | Usage |
|---|---|---|
| `--space-1` | 4px | Icon gaps, tight groupings |
| `--space-2` | 8px | Button padding, input gap |
| `--space-3` | 12px | Inline padding, small gaps |
| `--space-4` | 16px | Standard padding, section gap |
| `--space-6` | 24px | Section padding |
| `--space-8` | 32px | Page-level spacing |

## Border Radius

| Token | Value | Usage |
|---|---|---|
| `--radius-sm` | 4px | Buttons, inputs, small cards |
| `--radius-md` | 8px | Cards, modals, panels |
| `--radius-lg` | 12px | Feature cards |
| `--radius-full` | 9999px | Pills, avatars, badges |

## Shadows

| Token | Value | Usage |
|---|---|---|
| `--shadow-sm` | `0 1px 2px rgb(0 0 0 / 0.05)` | Resting elevation, dropdowns |
| `--shadow-md` | `0 4px 6px -1px rgb(0 0 0 / 0.1)` | Cards, drawers, modals |
| `--shadow-lg` | `0 10px 15px -3px rgb(0 0 0 / 0.1)` | Popovers, tooltips |

## Motion

- Default: 150ms ease-out-quart
- Layout: 200ms ease-out
- No bounce, no elastic, no spring overshoot
- `prefers-reduced-motion: reduce` → all transitions = 0

## Focus

```css
:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
```

Always present. Never removed. Replace browser defaults globally.

## Component Standards

### Buttons

- Height: 36px (compact), 40px (default)
- Horizontal padding: 12–16px
- Border-radius: `--radius-sm` (4px)
- States: default, hover (darken 8%), active (darken 12%), disabled (50% opacity, no pointer), loading (spinner + hidden text)
- Variants: primary (filled), secondary (border only), ghost (no border, text only), danger (red fill)

### Inputs

- Height: 40px
- Border: 1px solid `--color-border`
- Border-radius: `--radius-sm`
- Focus: border color becomes primary, focus ring appears
- Error: border becomes danger, helper text below
- Label always visible (not placeholder-only)

### Cards

- Background: `--color-background`
- Border: 1px solid `--radius-border` (or none if elevated)
- Border-radius: `--radius-md`
- Padding: `--space-4`
- Shadow: `--shadow-sm` (when elevated)
- No nested cards.

### Skeleton States

Loading areas use a pulsing gray background (`--color-background-subtle` → `--color-border` → `--color-background-subtle`), not spinners in content regions.

### Empty States

Always include: an icon or illustration, a headline explaining the state, a short description, and a primary action. Never a blank area or just "No items."

## Layout

- Max content width: 1200px, centered
- Page padding: `--space-6` desktop, `--space-4` mobile
- Header: fixed, full-width, 64px height
- Mobile breakpoint: 640px (hamburger nav below this)

## Anti-patterns to Avoid

- Side-stripe borders (left accent > 1px on cards)
- Gradient text
- Glassmorphism
- Hero-metric template (big number + small label)
- Identical card grids
- Modals as first navigation choice
- Pure `#000` or `#fff` neutrals
- Em dashes in UI copy