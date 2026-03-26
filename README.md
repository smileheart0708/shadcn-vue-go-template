# shadcn-vue-go-template

A full-stack SPA template with Vue 3 frontend and Go backend. The frontend serves as a modern UI layer with shadcn-vue-style components, while the Go backend provides REST API and serves the built frontend assets.

## Tech Stack

- **Frontend**: Vue 3 + TypeScript + Vite + TailwindCSS v4 + Pinia + Vue Router
- **Backend**: Go 1.26+ with SQLite (modernc.org/sqlite)
- **Package Manager**: pnpm (frontend), Go modules (backend)

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 18+
- pnpm

### Development

```bash
# Frontend development (port 5173)
cd web && pnpm dev

# Backend development (build frontend first so assets can be embedded)
cd web && pnpm build
cd ..
go run .
```

### Frontend-Only Mock Development

`MSW` can mock the current auth API during Vite development so the frontend can run without the Go server.

```bash
cd web
cp .env.example .env.local
# Set VITE_API_MOCKING=true
pnpm dev
```

Mock mode only intercepts `POST /api/auth/login` and `GET /api/auth/me`.

- Demo email: `demo@example.com`
- Demo password: `demo123456`
- Mock token: `mock-access-token`

### Production Build

```powershell
# Windows
.\build.ps1

# Output: app.exe (single-file executable with embedded frontend assets)
```

## Project Structure

```
shadcn-vue-go-template/
├── web/                    # Vue 3 frontend
│   ├── src/
│   │   ├── components/    # UI components (shadcn-vue style)
│   │   ├── composables/   # Vue composables
│   │   ├── layouts/       # Page layouts
│   │   ├── lib/          # Utilities (api, auth, utils)
│   │   ├── middleware/   # Route middleware
│   │   ├── router/       # Vue Router config
│   │   ├── stores/       # Pinia stores
│   │   └── views/       # Page views
│   └── package.json
├── internal/              # Go backend
│   ├── config/           # Configuration
│   ├── database/        # Database layer
│   ├── httpapi/         # HTTP API handlers
│   └── logging/         # Logging setup
├── main.go               # Go entry point
└── build.ps1             # Build script
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATA_DIR` | `./.data` | Data directory for SQLite DB |
| `DB_NAME` | `data.db` | SQLite database filename |
| `API_REQUEST_LOG_ENABLED` | `false` | Enable backend API request logging |
| `JWT_SECRET` | generated in `DATA_DIR/.jwt_secret` | JWT signing secret. Environment variable wins; otherwise the app reuses the persisted file or generates a new high-entropy secret automatically. |
| `REFRESH_COOKIE_NAME` | derived from `JWT_SECRET` | Optional refresh cookie name override. Set a unique value per project when multiple localhost apps share the same host. |
| `VITE_API_MOCKING` | `false` | Enable frontend MSW auth mocks during Vite development |

## Commands

### Frontend

```bash
cd web
pnpm install              # Install dependencies
pnpm dev                  # Start dev server
pnpm build                # Build for production
pnpm typecheck            # TypeScript type checking
pnpm lint                 # Run ESLint with auto-fix
```

### Backend

```bash
go mod tidy               # Clean up Go dependencies
go build -tags=go_json    # Build Go binary
go test ./...             # Run all tests
go test -run TestName    # Run a single test
```

## Features

- JWT-based authentication (`/api/auth/login`, `/api/auth/me`)
- SPA fallback routing (non-API routes return `index.html`)
- Gzip compression for static assets
- Frontend build artifacts embedded into the Go binary
- SQLite database with modernc.org/sqlite
- shadcn-vue-style UI components (sidebar, tabs, table, tooltip, sheet, switch, skeleton)
- Vue I18n plugin registration with typed locale resources under `web/src/locales/`

## I18n

The frontend now registers `vue-i18n` through `web/src/plugins/i18n.ts`.

- Locale resources live in `web/src/locales/`
- `zh-CN` is the default locale and `en-US` is the fallback locale
- Locale choice is restored from `localStorage` key `app.locale`, otherwise the browser locale is used when it matches a supported locale
- Shared text should be added under reusable namespaces such as `common.action`, `common.field`, `common.feedback`, and `common.state` before creating feature-specific keys
- When adding new translation keys, update every locale file together and reuse existing keys instead of creating near-duplicate labels

## Learn More

- [Vue 3 Documentation](https://vuejs.org/)
- [Go Documentation](https://go.dev/)
- [TailwindCSS v4](https://tailwindcss.com/)
