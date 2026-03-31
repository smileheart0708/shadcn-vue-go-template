# Loading Patterns

## Audit Commands

- `rg --files web/src/components/ui`
- `rg -n "@/components/ui/(alert-dialog|badge|button|card|dialog|empty|field|input|select|skeleton|spinner|table)" web/src`

Run both before inventing a new state wrapper or custom loader markup.

## Shared Component Inventory For Async States

- `Spinner`: `web/src/components/ui/spinner/Spinner.vue`
  Use for compact in-place pending indicators. The default shape is a `Loader2Icon`; usually only spacing or size classes are appropriate.
- `Skeleton`: `web/src/components/ui/skeleton/Skeleton.vue`
  Use for initial layout-preserving placeholders. Width and height classes are expected; avoid changing its default animation, color, or radius without a strong local reason.
- `Empty`: `web/src/components/ui/empty/*`
  Use for empty, blocked, or failed-first-load states. Prefer composing `EmptyHeader`, `EmptyTitle`, `EmptyDescription`, `EmptyContent`, and `EmptyMedia` instead of replacing the base look.
- `TableEmpty`: `web/src/components/ui/table/TableEmpty.vue`
  Use for empty table bodies instead of custom rows or detached empty banners.
- `Button`: `web/src/components/ui/button/Button.vue`
  Prefer `variant` and `size` props over class-based shape changes.
- `Card`: `web/src/components/ui/card/*`
  Use as the stable shell around skeletons, settings panels, and retry states.
- `Dialog` and `AlertDialog`: `web/src/components/ui/dialog/*`, `web/src/components/ui/alert-dialog/*`
  Use for modal loading and submit states; keep spinners in action rows instead of covering the whole dialog.
- `Field`: `web/src/components/ui/field/*`
  Use for form layout so loading and pending states stay aligned with the final form structure.
- `Table`: `web/src/components/ui/table/*`
  Use skeleton rows during initial load and `TableEmpty` when the loaded dataset is empty.

Note: `web/src/components/ui/progress/` exists as a directory today, but it does not currently expose an implemented shared progress primitive. Verify actual files before depending on it.

## Style Guardrails

- Default to the shared component's shipped appearance. Composition should create consistency; custom restyling should be the exception.
- Use classes on shared components mostly for layout concerns such as width, height, margin, gap, flex, grid, and visibility.
- Avoid overriding `rounded-*`, `border-*`, `bg-*`, `shadow-*`, or typography classes on shared components unless the user explicitly asks for a visual change or an existing local pattern already relies on that adjustment.
- If a state needs a different silhouette, check whether the surrounding container should change instead of the shared component itself.

## Repository Examples

- `web/src/views/SystemLogsView.vue`
  Uses `Skeleton` rows for the initial log connection and keeps loaded content visible while later reconnect actions show inline pending state.
- `web/src/views/AdminUsersView.vue`
  Combines `Table` + `Skeleton` for initial load, `Empty` for failed first load, `TableEmpty` for a loaded-but-empty result, and inline `Spinner` usage for refresh and destructive action confirmation.
- `web/src/views/AdminSettingsView.vue`
  Uses `Card` shells around skeleton blocks for first load, an `Empty` retry state, and a button-level `Spinner` during save.
- `web/src/components/register/RegisterForm.vue`
  Shows three distinct states with shared primitives: skeleton-first load, card retry state, and `Empty` for disabled registration. The submit action uses an inline `Spinner`.
- `web/src/components/admin-users/AdminUserDialog.vue`
  Uses `Dialog` and `Field` composition with button-level pending UI instead of custom modal overlays.

## Pattern Notes

### Page Or Panel Initial Load

- Keep the final layout shell visible when possible.
- Prefer several small skeleton blocks that resemble the final structure over one generic full-width rectangle.

### Lists, Tables, And Logs

- Mirror row count and cell density with `Skeleton`.
- Switch to `TableEmpty` only after data has actually loaded.

### Forms And Dialogs

- Keep labels, descriptions, and button placement on shared `Field*` and `Dialog*` components.
- Disable only the action that is pending.

### Retry And Refresh

- Use a retry `Button` inside the local `Empty` or panel footer.
- For background refresh, keep current data on screen and surface the pending state inline.

## Validation

- `cd web && pnpm lint`
- `cd web && pnpm typecheck`
- `cd web && pnpm build`
