---
name: shadcn-async-states
description: Design or refactor async UI states in this Vue 3.5 + Tailwind 4 + shadcn-vue repository. Use when editing `web/src` pages or components that fetch data, defer route content, submit forms, refresh sections, paginate tables, or need loading, empty, error, retry, or pending states. Audit `web/src/components/ui` first, prefer installed shared UI components and existing local patterns over ad hoc wrappers or custom CSS, and default to preserving the shared components' default visual styles.
---

# Shadcn Loading States

## Goals

- Reuse repository UI primitives for loading, empty, error, retry, and pending states.
- Keep state ownership local to the page, panel, dialog, or control that owns the async boundary.
- Preserve the default look of shared UI components unless the task explicitly calls for a visual change.

## Audit First

- List available shared components with `rg --files web/src/components/ui`.
- Search for existing usage patterns with `rg -n "@/components/ui/(alert-dialog|badge|button|card|dialog|empty|field|input|select|skeleton|spinner|table)" web/src`.
- Treat actual exported files and current imports as the source of truth. Do not assume a folder name means the component is implemented and ready.
- Reuse an existing composition before introducing a new wrapper, helper, or custom state component.

## Preferred Primitives In This Repository

- Use `Spinner` for compact indeterminate pending indicators.
- Use `Skeleton` for initial placeholders that preserve the final layout.
- Use `Empty` with `EmptyHeader`, `EmptyTitle`, `EmptyDescription`, `EmptyContent`, and `EmptyMedia` for stand-alone empty or failed-first-load states.
- Use `TableEmpty` inside tables instead of custom empty rows.
- Use shared structure components such as `Button`, `Card`, `Dialog`, `AlertDialog`, `Field`, `Input`, `Select`, `Badge`, and `Table` to compose stateful UIs instead of building one-off markup.

## Choose The Right Pattern

### Initial Data Load

- Render `Skeleton` inside the final shell, such as `Card`, `Table`, form groups, or a panel container.
- Match the final widths, heights, and row counts closely enough to avoid visible layout jump.
- Keep surrounding chrome visible when it is already known and stable.

### Background Refresh Or Retry

- Keep previously loaded content visible.
- Put `Spinner` in the triggering button, summary row, dialog footer, or status line.
- Disable only the relevant action while pending.

### Empty, No Results, Or Failed First Load

- Use `Empty` for stand-alone pages, panels, and first-load failures with a retry CTA.
- Use `TableEmpty` inside `TableBody` for empty datasets.
- Pair the state with a `Button` when the user can retry, create, or navigate.

### Forms And Dialogs

- Keep form structure on shared `Field*`, `Dialog*`, and `Button` components.
- Show submit pending state by placing `Spinner` inside the action button and disabling that action.
- If form data itself is loading before first render, skeleton the field group or card instead of flashing an empty form that later fills in.

### Determinate Progress

- Prefer a shared repository component if one exists at the time of the task.
- If no shared determinate progress primitive exists, do not invent a one-off progress bar unless the task explicitly needs it.

## Styling Guardrails

- Prefer shared component props, variants, and slots before reaching for custom classes.
- Use `class` on shared components mainly for layout, spacing, sizing, and conditional visibility.
- Do not override shared component defaults for border radius, border style, background, shadow, or text treatment unless the user explicitly asks for a visual change, an existing repository pattern already does it nearby, or the component API exposes a supported variant for it.
- Avoid creating custom loader markup or wrapper components just to change component shape.
- Prefer `variant` and `size` props on `Button` over ad hoc overrides.
- For `Skeleton`, sizing classes are expected. Do not restyle its default color, animation, or radius unless an established repository pattern requires it.
- For `Empty`, prefer composition with `EmptyMedia`, `EmptyHeader`, and `EmptyContent` over stripping its default border or padding. Only override those defaults to preserve an existing surrounding container pattern.

## Accessibility And Copy

- Keep loading and empty states separate. A skeleton is not an empty state.
- Add accessible text when spinner-only UI would otherwise be silent.
- Avoid duplicated signals such as a spinner plus repeated "Loading..." copy when one clear signal is enough.
- Preserve focus management and clear disabled states for pending actions.

## Validation

- Run `pnpm lint`, `pnpm typecheck`, and `pnpm build` in `web/` after changing async UI states.

## References

- Read `references/patterns.md` for current project component inventory, style guardrails, and repository examples.
