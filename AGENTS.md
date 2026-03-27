# Repository Guidelines

## Project Structure & Module Organization
- `main.go` starts the Go server; `frontend_embed.go` embeds the built frontend into the binary.
- `internal/` holds backend code: `auth`, `config`, `database`, `httpapi`, `logging`, and `users`.
- `web/src/` contains the Vue 3 app. Key areas are `components/`, `views/`, `layouts/`, `router/`, `stores/`, `lib/`, `plugins/`, `mocks/`, and `locales/`.
- `web/src/components/ui/` contains reusable shadcn-vue style primitives. `web/dist/` is generated output and should not be edited.

## Build, Test, and Development Commands
- `cd web && pnpm install` installs frontend dependencies.
- `cd web && pnpm dev` starts the Vite dev server.
- `cd web && pnpm build` builds the frontend for production.
- `cd web && pnpm lint` runs ESLint; `cd web && pnpm typecheck` runs Vue/TypeScript checks.
- `go test ./...` runs all Go tests.
- `go build -tags=go_json .` builds the backend binary.
- `.\build.ps1` builds the frontend, then the Go app into `app.exe`.

## Coding Style & Naming Conventions
- Use `gofmt` for Go code and keep packages focused and small.
- Frontend formatting follows Prettier: no semicolons, single quotes, 200 character print width, trailing commas, and LF line endings.
- Follow existing Vue naming patterns: components in `PascalCase.vue`, composables as `use*.ts`, and route/store files with descriptive lowercase names.
- Keep translation keys grouped logically in `web/src/locales/` and reuse shared namespaces such as `common.*`.
- For new page or component loading states, add `Skeleton` locally at the relevant view or subcomponent instead of trying to auto-register or globally inject it. Keep the dependency explicit where the loading UI is needed.

## Testing Guidelines
- Go tests live beside the code as `*_test.go` files.
- There is no separate frontend test runner configured; use `pnpm lint`, `pnpm typecheck`, and `pnpm build` to validate UI changes.
- Add or update tests for backend changes in `config`, `httpapi`, `logging`, or `users` when behavior changes.

## Commit & Pull Request Guidelines
- Commit history uses scoped prefixes like `feat(web): ...`, `fix(backend): ...`, and `style(web): ...`.
- PRs should describe the change, list validation commands run, and include screenshots for UI updates when relevant.
- Link related issues when available and call out any migration or config impact.

## Security & Configuration Tips
- Do not commit `.env`, `.data/`, `node_modules/`, `web/dist/`, or built executables.
- Runtime data lives under `.data/`; the JWT secret is loaded from `JWT_SECRET` or generated there automatically.
- Frontend auth mocking is controlled by `VITE_API_MOCKING=true` in `web/.env.local`.
