# AGENTS.md - Developer Guide for shadcn-vue-go-template

## Project Overview

This is a full-stack SPA template with Vue 3 frontend and Go backend. The frontend serves as a modern UI layer with shadcn-vue-style components, while the Go backend provides REST API and serves the built frontend assets.

## Tech Stack

- **Frontend**: Vue 3 + TypeScript + Vite + TailwindCSS v4 + Pinia + Vue Router
- **Backend**: Go 1.26+ with SQLite (modernc.org/sqlite)
- **Package Manager**: pnpm (frontend), Go modules (backend)
- **UI Components**: Custom shadcn-vue-style components (sidebar, tabs, table, tooltip, sheet, switch, skeleton)

## Build Commands

### Frontend (web/)

```bash
cd web
pnpm install              # Install dependencies
pnpm dev                  # Start dev server (port 5173)
pnpm build                # Build for production
pnpm preview              # Preview production build
pnpm typecheck            # TypeScript type checking
pnpm lint                 # Run ESLint with auto-fix
```

### Backend (root/)

```bash
go mod tidy               # Clean up Go dependencies
go build -tags=go_json    # Build Go binary
go test ./...             # Run all tests
go test -v ./internal/... # Run tests with verbose output
go test -run TestName    # Run a single test
```

### Full Build

```powershell
.\build.ps1               # Build frontend + backend (Windows)
```

The build script outputs `app.exe` in the project root.

## Code Style Guidelines

### General

- Use Prettier for code formatting (configured in VSCode)
- Run `pnpm lint --fix` before committing frontend code
- Run `gofmt -s` or `go fmt` for Go code formatting

### Frontend (Vue 3 + TypeScript)

#### Imports

- Use path aliases: `@/` maps to `web/src/`
- Order imports: Vue/Vue Router/Pinia -> External libs -> Internal imports -> Types
- Example:
  ```typescript
  import { ref, computed } from 'vue'
  import { useRouter } from 'vue-router'
  import { useAuthStore } from '@/stores/auth'
  import { cn } from '@/lib/utils'
  import type { User } from '@/types'
  ```

#### Naming Conventions

- **Components**: PascalCase (`Sidebar.vue`, `UserProfile.vue`)
- **Composables**: camelCase with `use` prefix (`useAuth.ts`, `useUser.ts`)
- **Stores**: PascalCase (`auth.ts` -> `AuthStore`)
- **Utils**: camelCase (`formatDate.ts`, `apiClient.ts`)
- **Types/Interfaces**: PascalCase (`UserResponse`, `LoginRequest`)

#### Vue Component Patterns

- Use `<script setup lang="ts">` for all components
- Use TypeScript for all props and emits
- Extract types to `src/types/` directory
- Use composables for reusable logic
- Use Pinia for global state management

#### CSS/Tailwind

- Use TailwindCSS utility classes
- Use `cn()` from `@/lib/utils` for conditional classes
- Custom styles in `src/style.css` (global)
- Component-scoped styles when needed

### Backend (Go)

#### Package Structure

```
internal/
  config/      # Configuration
  database/   # Database layer
  handlers/   # HTTP handlers (reserved for expansion)
  httpapi/    # Main HTTP API
  logging/    # Logging setup
  middleware/ # HTTP middleware
```

#### Naming Conventions

- **Functions**: PascalCase (`NewHandler`, `OpenDatabase`)
- **Variables**: camelCase (`logger`, `dbPath`)
- **Constants**: PascalCase or camelCase for package-level constants
- **Error variables**: `Err*` prefix for sentinel errors

#### Error Handling

- Use `slog` for structured logging
- Return errors with context using `fmt.Errorf("context: %w", err)`
- Handle errors early, return early pattern

#### HTTP API

- Use standard library `net/http`
- JSON responses via `json.go` helpers
- Middleware chain pattern in `middleware.go`

## Testing

### Frontend

No test framework configured yet. Consider adding Vitest:
```bash
pnpm add -D vitest
pnpm vitest run          # Run tests once
pnpm vitest run specific # Run specific test file
```

### Backend

Go uses standard `testing` package:
```bash
go test -v ./internal/httpapi/...
go test -run TestHealthz ./internal/httpapi/
```

- Use `t.Parallel()` for parallelizable tests
- Use `t.Helper()` for test helper functions
- Use `httptest` for HTTP handler testing

## Development Workflow

1. **Frontend Development**: `cd web && pnpm dev`
2. **Backend Development**: Run `go run .` in root (requires frontend built to `web/dist/`)
3. **API Proxy**: Frontend Vite config proxies `/api` to `http://localhost:8080`

## Environment Variables

### Backend

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATA_DIR` | `./.data` | Data directory for SQLite DB |
| `DB_NAME` | `app.db` | SQLite database filename |
| `API_REQUEST_LOG_ENABLED` | `false` | Enable backend API request logging |
| `JWT_SECRET` | `dev-secret-change-in-prod` | JWT signing secret |
| `FRONTEND_DIST_DIR` | `./web/dist` | Frontend assets directory |

## VSCode Extensions

Recommended extensions (see `.vscode/extensions.json`):
- Vue.volar
- vitest.explorer
- dbaeumer.vscode-eslint
- EditorConfig.EditorConfig
- oxc.oxc-vscode
- esbenp.prettier-vscode

## Notes

- Frontend assets (after build) are served by Go backend
- SPA fallback: non-API routes return `index.html` for client-side routing
- Gzip compression enabled for static assets
- JWT-based authentication with `/api/auth/login` and `/api/auth/me`
