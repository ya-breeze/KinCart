# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

KinCart is an intelligent family shopping assistant that connects users into Families with synchronized data. It supports two primary roles:
- **Manager**: Plans shopping lists, sets budgets, and configures store layouts
- **Shopper**: Executes shopping in-store with optimized routes and real-time updates

Tech Stack: Go (Gin) backend, React (Vite) frontend, SQLite (GORM), Docker deployment.

## Build & Run Commands

### Development Workflow
```bash
# Build entire project
make build

# Run tests (includes backend, frontend, and E2E)
make test

# Run individual test suites
make test-backend        # Go unit tests
make test-frontend       # React/Vitest tests
make test-e2e           # Playwright E2E tests (auto-starts Docker if needed)

# Lint code
make lint               # Both backend and frontend
make lint-backend       # golangci-lint
make lint-frontend      # ESLint

# Docker operations
make docker-up          # Start all containers
make docker-down        # Stop containers
make docker-up-e2e      # Start with E2E test assets
```

### Testing Credentials
From `.cursorrules`:
- Manager: `manager_test` / `pass1234`
- Shopper: `shopper_test` / `pass1234`

Seed test data: `make seed-test-data`

### User Management
```bash
# Add family
make add-family NAME="TheSmiths"

# Add user to family
make add-user FAMILY="TheSmiths" UNAME="john" PASS="mypassword"
```

### Backend Development
```bash
cd backend

# Run server locally (port 8080)
go run cmd/server/main.go

# Run admin CLI
go run cmd/admin/main.go add-user --family "TestFamily" --username "alice" --password "pass123"

# Run specific test
go test ./internal/handlers -run TestGetLists

# Build binaries
go build -o bin/server cmd/server/main.go
go build -o bin/admin cmd/admin/main.go
```

### Frontend Development
```bash
cd frontend

# Dev server (port 5173)
npm run dev

# Build for production
npm run build

# Run specific test
npm test -- LazyImage.test.jsx
```

## Architecture

### Backend Structure (`backend/`)

**Entry Points:**
- `cmd/server/main.go` - HTTP server with Gin routing, middleware setup, background jobs
- `cmd/admin/main.go` - CLI for user/family management

**Core Packages:**
- `internal/models/` - GORM models that embed shared `kin-core` models for multi-tenancy
- `internal/database/` - DB initialization, migrations, environment-based seeding
- `internal/handlers/` - HTTP request handlers (auth, lists, items, categories, shops, flyers, receipts)
- `internal/middleware/` - Auth (JWT), CORS, rate limiting, upload security
- `internal/services/` - Business logic (receipts, file storage)
- `internal/ai/` - Gemini client for receipt parsing and flyer processing
- `internal/flyers/` - Flyer download scheduler, parser, manager
- `internal/utils/` - Shared utilities

**Key Architectural Patterns:**

1. **Multi-Tenancy via kin-core**: Models embed `coremodels.TenantModel` which includes `FamilyID`. All queries must be scoped by family to ensure data isolation. Use the `github.com/ya-breeze/kin-core/db` package's scoping utilities (e.g., `db.ScopedQuery`, `db.ScopedFirst`) to automatically filter by `FamilyID` from context.

2. **Family ID Context**: The `AuthMiddleware` extracts the JWT, loads the user, and sets `c.Set("family_id", user.FamilyID)` in the Gin context. Handlers retrieve this with `c.GetUint("family_id")`.

3. **Database Migration Strategy**: On startup, `database.InitDB()` performs:
   - Manual `ALTER TABLE` for `family_id` columns on existing tables (SQLite workaround)
   - Auto-migration via GORM for all models
   - Environment-based seeding (`KINCART_SEED_USERS`, `KINCART_SEED_FLYERS`)

4. **Background Jobs**:
   - `middleware.CleanupBlacklist()` - JWT token blacklist cleanup
   - `flyers.StartScheduler()` - Flyer download/parsing (if `GEMINI_API_KEY` set)
   - Receipt processing ticker (every 10 minutes)

5. **Security Middleware**:
   - `CORSMiddleware()` - Origin validation based on `ALLOWED_ORIGINS` env var
   - `LoginRateLimiter()` - 5 req/min per IP on `/api/auth/login`
   - `UploadSecurityMiddleware()` - Content-Type validation, path traversal protection
   - `AuthMiddleware()` - JWT validation, token blacklist check

6. **File Storage**: All uploads go to `UPLOADS_PATH` (default `/data/uploads` in Docker). Flyer items stored in `FLYER_ITEMS_PATH` (default `/data/flyer_items`).

### Frontend Structure (`frontend/src/`)

**Pages:**
- `Dashboard.jsx` - Main view for lists
- `ListDetail.jsx` - Individual list with items, grouped by category
- `SettingsPage.jsx` - Category/shop management
- `FlyerItemsPage.jsx` - Browse discounted items from flyers
- `FlyerStatsPage.jsx` - Flyer statistics dashboard
- `LoginPage.jsx` - Authentication

**Components:**
- `LazyImage.jsx` - Lazy-loaded images with fade-in
- `ImageModal.jsx` - Full-screen image viewer
- `ReceiptUploadModal.jsx` - Receipt upload with camera/file support

**Context:**
- `context/AuthContext.jsx` - User authentication state

**Styling:**
- Vanilla CSS only (no CSS frameworks per `.cursorrules`)
- `index.css` - Global styles
- `App.css` - App-level styles

### Database Models

**Multi-Tenant Models** (include `FamilyID`):
- `ShoppingList` - Lists with status (planning/in-progress/completed)
- `Item` - Individual shopping items with category, price, photo, flyer/receipt linking
- `Category` - User-defined categories with sort order
- `Shop` - User-defined shops
- `Receipt` - Uploaded receipts with parsed items
- `ItemFrequency` - Tracks commonly purchased items

**Global Models** (no `FamilyID`):
- `Family` - Tenant boundary with currency setting
- `User` - Belongs to a family, has username/password
- `Flyer` - Store promotional flyers with pages
- `FlyerPage` - Individual flyer pages (images)
- `FlyerItem` - Parsed items from flyers
- `ShopCategoryOrder` - Per-shop category routing order
- `JobStatus` - Background job tracking

### API Routes (`cmd/server/main.go`)

**Public:**
- `POST /api/auth/login` - Login with rate limiting

**Protected** (require JWT):
- `GET /api/auth/me` - Current user info
- `POST /api/auth/logout` - Logout (blacklists token)
- `GET /api/lists` - List all shopping lists for family
- `POST /api/lists` - Create new list
- `PATCH /api/lists/:id` - Update list
- `POST /api/lists/:id/duplicate` - Clone list
- `DELETE /api/lists/:id` - Delete list
- `POST /api/lists/:id/items` - Add item to list
- `POST /api/lists/:id/receipts` - Upload receipt
- `PATCH /api/items/:id` - Update item
- `DELETE /api/items/:id` - Delete item
- `POST /api/items/:id/photo` - Upload item photo
- `GET /api/categories` - Get categories
- `POST /api/categories` - Create category
- `PATCH /api/categories/:id` - Update category
- `DELETE /api/categories/:id` - Delete category
- `PATCH /api/categories/reorder` - Reorder categories
- `GET /api/family/config` - Get family config
- `PATCH /api/family/config` - Update family config
- `GET /api/family/frequent-items` - Get frequently purchased items
- `GET /api/shops` - Get shops
- `POST /api/shops` - Create shop
- `PATCH /api/shops/:id` - Update shop
- `DELETE /api/shops/:id` - Delete shop
- `GET /api/shops/:id/order` - Get category order for shop
- `PATCH /api/shops/:id/order` - Set category order for shop
- `GET /api/flyers/*` - Flyer-related endpoints

**Internal** (blocked by Nginx):
- `POST /api/internal/flyers/parse` - Trigger flyer parsing
- `POST /api/internal/flyers/download` - Trigger flyer download

## Environment Variables

**Required:**
- `JWT_SECRET` - JWT signing key (change in production!)
- `ALLOWED_ORIGINS` - Comma-separated CORS origins (e.g., `https://kincart.example.com`)

**Optional:**
- `DB_PATH` - SQLite database path (default: `./data/kincart.db`)
- `UPLOADS_PATH` - Upload directory (default: `./uploads`)
- `FLYER_ITEMS_PATH` - Flyer items directory (default: `./uploads/flyer_items`)
- `KINCART_DATA_PATH` - Data directory for Docker volume
- `KINCART_SEED_USERS` - Auto-create users on startup: `Family:user:pass,Family2:user2:pass2`
- `KINCART_SEED_FLYERS` - Seed flyer data: `Shop:Item1|Price1,Item2|Price2`
- `GEMINI_API_KEY` - Google Gemini API key for AI features (receipt parsing, flyer parsing)
- `NGINX_HTTP_PORT` - Nginx HTTP port (default: 80)
- `NGINX_HTTPS_PORT` - Nginx HTTPS port (default: 443)

## Testing Strategy

### Backend Tests
Located in `backend/internal/handlers/*_test.go`:
- `auth_test.go` - Login, logout, auth middleware
- `items_test.go` - Item CRUD operations
- `lists_test.go` - List CRUD operations
- `isolation_test.go` - **Critical**: Tests multi-tenant data isolation
- `utils_test.go` - Helper utilities

Test pattern uses `SetupTestDB()` to create isolated in-memory SQLite instances.

### Frontend Tests
Located in `frontend/src/**/*.test.jsx`:
- `Dashboard.test.jsx` - Dashboard rendering
- `LoginPage.test.jsx` - Login flow
- `LazyImage.test.jsx` - Lazy loading behavior

Uses Vitest + React Testing Library + happy-dom.

### E2E Tests
Located in `e2e/tests/`:
- Playwright-based browser tests
- `docker-compose.e2e.yml` includes test assets
- Auto-started by `make test-e2e` if not running

## Important Development Notes

1. **Multi-Tenant Isolation**: Always scope queries by `family_id`. The `isolation_test.go` file validates this. When adding new endpoints, ensure they use the family ID from context.

2. **Styling**: Use vanilla CSS only per `.cursorrules`. No CSS frameworks or component libraries.

3. **Security**:
   - Never commit secrets to `.env` files
   - All uploads must go through `UploadSecurityMiddleware`
   - CORS origins must be explicitly configured in production
   - JWT tokens are blacklisted on logout

4. **Database Changes**: After modifying models, restart the backend to trigger auto-migration. For SQLite NOT NULL column additions, see the manual ALTER TABLE pattern in `database/db.go`.

5. **Flyer & Receipt Features**: Require `GEMINI_API_KEY` to be set. These are optional features that gracefully degrade if the API key is missing.

6. **Docker Deployment**: The app runs behind Nginx reverse proxy. Internal API routes (`/api/internal/*`) should only be accessible within the Docker network.

7. **Development Plan**: See `development_plan.md` for feature roadmap and implementation status.
