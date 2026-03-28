# Loading Patterns

## Existing Components

- `Spinner`: `web/src/components/ui/spinner/Spinner.vue`
- `Skeleton`: `web/src/components/ui/skeleton/Skeleton.vue`

Use these exact local components. Do not add a parallel loader primitive.

## Existing Repository Examples

- `web/src/views/SystemLogsView.vue`
  Uses `Skeleton` rows during the first stream connection, then keeps the content area visible and switches the reconnect action to an inline spinning icon for later connection attempts.
- `web/src/components/app/RouteLoadingBar.vue`
  Provides the repository-wide route transition indicator. Treat this as the route-level exception instead of replacing it with generic page spinners.

## Suggested Patterns

### Page Or Panel Initial Load

- Render skeletons that approximate the final structure.
- Keep dimensions close to the loaded UI so layout shift stays low.
- Prefer several small skeleton blocks over one oversized rectangle when the final content has clear internal structure.

### Lists, Tables, And Logs

- Mirror expected row count with `Skeleton` rows.
- Use consistent heights, border radius, and spacing with the eventual rows.
- Keep filters and toolbars visible if they are already usable during load.

### Forms And Dialogs

- Use `Spinner` inside submit buttons or action footers.
- Disable the relevant action while pending instead of blanketing the whole screen.
- If form data itself is loading before first render, skeleton the field group rather than showing an empty form that suddenly fills in.

### Refresh, Retry, And Polling

- Keep already loaded content visible.
- Put `Spinner` in the specific control or status area that initiated the refresh.
- Reserve skeletons for cases where content is genuinely unavailable, not merely being updated.
