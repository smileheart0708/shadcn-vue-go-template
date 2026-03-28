---
name: shadcn-loading-states
description: Prefer project-standard loading UI in this Vue 3.5 + shadcn-vue repository. Use when creating or editing frontend pages and components under web/src that fetch async data, defer route content, submit forms, refresh panels, or otherwise need loading, pending, or empty-to-loaded states. Prefer the local shadcn Spinner for compact in-place activity indicators and the local shadcn Skeleton for first-load placeholders that preserve layout.
---

# Shadcn Loading States

## Overview

Use the repository's existing loading primitives instead of ad hoc text, CSS loaders, or new dependencies.
Keep loading UI local to the page or subcomponent that owns the async boundary.

## Quick Start

- Import `Spinner` from `@/components/ui/spinner` for compact pending states.
- Import `Skeleton` from `@/components/ui/skeleton` for placeholder layouts during initial fetches.
- Match the loading UI to the final layout instead of using generic centered text when the shape matters.
- Reuse both when appropriate: `Skeleton` for first paint, `Spinner` for later refresh or button-level actions.

## Choose The Right Primitive

Use `Skeleton` when:
- Content is not ready yet and the final layout is already known.
- The user would benefit from stable page structure during the first load.
- Rendering cards, tables, lists, detail panes, dashboards, or form sections that should not jump.

Use `Spinner` when:
- An action is already in context and only needs a compact progress indicator.
- Disabling a button, refreshing a section, reconnecting a stream, or waiting inside a dialog.
- The layout should remain visible and interactive context should stay obvious.

Use both when:
- A page has an initial blocking load plus a later non-blocking refresh action.
- A section skeletons on first load, then uses spinner-bearing controls for retries, polling, or export actions.

Do not:
- Add a new loading library.
- Replace a layout-preserving skeleton with a tiny spinner when content jump would get worse.
- Use large full-page spinners by default when local skeletons or inline spinners communicate state better.
- Hide all existing content behind a blocking overlay for routine refreshes.

## Placement Rules

- Keep the indicator near the async boundary that owns the state.
- For page-level initial loads, render skeletons inside the view or subpanel, not as a global auto-injected fallback.
- For table or log panes, mirror row height and count with skeleton rows.
- For button actions, keep the label readable and pair the spinner with `disabled` or `aria-busy` as needed.
- For route transitions, preserve the repository's dedicated route-loading UI unless the task explicitly changes that behavior.

## Accessibility And Copy

- When spinner-only UI would otherwise be visual-only, include accessible loading text such as `common.state.loading` in an `sr-only` element or equivalent label.
- Do not show both a spinner and repeated "Loading..." copy when one clear signal is enough.
- Preserve empty states separately from loading states. Skeletons are for not-yet-loaded content, not no-data cases.

## Repository Conventions

- Follow the repository rule to add loading states locally at the relevant view or subcomponent.
- Prefer existing UI building blocks from `web/src/components/ui/` before inventing custom wrappers.
- Keep imports explicit; do not globally register loading components.
- Match this repository's modern Vue patterns and current styling conventions.

## References

- Read `references/patterns.md` for repository-specific examples and recommended placements.

## Resources (optional)

Create only the resource directories this skill actually needs. Delete this section if no resources are required.

### scripts/
Executable code (Python/Bash/etc.) that can be run directly to perform specific operations.

**Examples from other skills:**
- PDF skill: `fill_fillable_fields.py`, `extract_form_field_info.py` - utilities for PDF manipulation
- DOCX skill: `document.py`, `utilities.py` - Python modules for document processing

**Appropriate for:** Python scripts, shell scripts, or any executable code that performs automation, data processing, or specific operations.

**Note:** Scripts may be executed without loading into context, but can still be read by Codex for patching or environment adjustments.

### references/
Documentation and reference material intended to be loaded into context to inform Codex's process and thinking.

**Examples from other skills:**
- Product management: `communication.md`, `context_building.md` - detailed workflow guides
- BigQuery: API reference documentation and query examples
- Finance: Schema documentation, company policies

**Appropriate for:** In-depth documentation, API references, database schemas, comprehensive guides, or any detailed information that Codex should reference while working.

### assets/
Files not intended to be loaded into context, but rather used within the output Codex produces.

**Examples from other skills:**
- Brand styling: PowerPoint template files (.pptx), logo files
- Frontend builder: HTML/React boilerplate project directories
- Typography: Font files (.ttf, .woff2)

**Appropriate for:** Templates, boilerplate code, document templates, images, icons, fonts, or any files meant to be copied or used in the final output.

---

**Not every skill requires all three types of resources.**
