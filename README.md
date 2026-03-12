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

# Backend development (requires frontend built to web/dist/)
go run .
```

### Production Build

```powershell
# Windows
.\build.ps1

# Output: app.exe
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
| `DB_NAME` | `app.db` | SQLite database filename |
| `API_REQUEST_LOG_ENABLED` | `false` | Enable backend API request logging |
| `JWT_SECRET` | `dev-secret-change-in-prod` | JWT signing secret |
| `FRONTEND_DIST_DIR` | `./web/dist` | Frontend assets directory |

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
- SQLite database with modernc.org/sqlite
- shadcn-vue-style UI components (sidebar, tabs, table, tooltip, sheet, switch, skeleton)

## Learn More

- [Vue 3 Documentation](https://vuejs.org/)
- [Go Documentation](https://go.dev/)
- [TailwindCSS v4](https://tailwindcss.com/)
