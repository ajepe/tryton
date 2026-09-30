# Talon: Go Rewrite of Tryton — Master Plan

**Project name**: Talon — fast, sharp, precise.

## Goal

Rewrite the entire Tryton project (trytond server, 221 modules, clients, reports) in Go while maintaining compatibility with the existing PostgreSQL schema and JSON-RPC protocol.

## Compatibility Target

- **Database**: Same PostgreSQL schema (tables, columns, constraints, sequences)
- **Protocol**: JSON-RPC with context-as-last-arg, tagged codecs, X-Tryton-Cache
- **Data**: Existing Tryton databases must be readable/writable by the Go server
- **Modules**: Python modules continue to work during transition (dual-run)

## Current State

| Component | Language | LOC | Notes |
|-----------|----------|-----|-------|
| trytond core | Python | ~78k | ORM, Pool, schema sync, RPC/WSGI, actions, security, i18n, bus, wizards, reports |
| modules | Python + XML | ~194k + 2,223 XML | 221 modules, 388 test files |
| sao (web client) | JavaScript | ~30k | jQuery/Bootstrap, server-driven UI |
| tryton (desktop) | Python | ~34k | PyGObject/GTK |
| proteus/naiad | Python | ~5k | Client libraries |
| Tests | Python | 461 files | Core + module tests |

## Phases

### Phase 0: Contract + Conformance Harness
**Issue**: [#2](https://github.com/ajepe/tryton/issues/2)
**Effort**: 2-4 person-weeks

Define the exact compatibility target and build a differential testing harness that runs Python trytond and the Go core side-by-side on identical database dumps, comparing RPC outputs.

**Deliverables**:
- `docs/go-rewrite/compatibility-contract.md`
- `conformance/` — differential test runner
- `conformance/cases/` — test cases derived from existing Python tests

### Phase 1: Go Core Engine
**Issue**: [#3](https://github.com/ajepe/tryton/issues/3)
**Effort**: 4-8 person-weeks

Build the foundational Go engine: metadata registry, field types, ORM with batched hydration, and PostgreSQL schema synchronization/migration.

**Deliverables**:
- `go/` — Go module root
- `go/internal/metadata/` — model/field registry
- `go/internal/orm/` — ORM engine
- `go/internal/schema/` — schema sync + migrations
- `go/internal/backend/postgresql/` — PostgreSQL driver

### Phase 2: Request Layer
**Issue**: [#4](https://github.com/ajepe/tryton/issues/4)
**Effort**: 4-8 person-weeks

Build the HTTP/RPC layer: session management, JSON-RPC protocol, security rules, internationalization, XML data loader, and PYSON expression evaluator.

**Deliverables**:
- `go/internal/rpc/` — JSON-RPC protocol
- `go/internal/session/` — session/auth
- `go/internal/security/` — access rules
- `go/internal/i18n/` — translations
- `go/internal/xml/` — XML data loader
- `go/internal/pyson/` — PYSON evaluator
- `go/internal/action/` — action system

### Phase 3: Module Ports
**Issue**: [#5](https://github.com/ajepe/tryton/issues/5)
**Effort**: 20-40 person-weeks

Port the core business modules from Python to Go, in dependency order, with full test coverage derived from the existing 388 module test files.

**Module Port Order**:
1. ir — internal resource management
2. res — users, groups, access rules
3. country — countries, subdivisions, postal codes
4. party — parties, categories, addresses, contact mechanisms
5. product — products, templates, categories, units of measure
6. account — chart of accounts, fiscal years, periods, moves, lines
7. stock — warehouses, shipments, moves, inventory
8. sale — sale orders, lines, shipments, invoices
9. purchase — purchase orders, lines, shipments, invoices

**Deliverables**:
- `go/internal/modules/ir/`
- `go/internal/modules/res/`
- `go/internal/modules/country/`
- `go/internal/modules/party/`
- `go/internal/modules/product/`
- `go/internal/modules/account/`
- `go/internal/modules/stock/`
- `go/internal/modules/sale/`
- `go/internal/modules/purchase/`

### Phase 4: Platform Services
**Issue**: [#6](https://github.com/ajepe/tryton/issues/6)
**Effort**: 4-8 person-weeks

Build the platform-level services that modules depend on: report generation, wizard engine, scheduled tasks, file storage, email, and the notification bus.

**Deliverables**:
- `go/internal/report/` — report engine
- `go/internal/wizard/` — wizard engine
- `go/internal/cron/` — scheduler
- `go/internal/filestore/` — file storage
- `go/internal/email/` — email service
- `go/internal/bus/` — notification bus

### Shared Infrastructure

#### Shared Client Library
**Issue**: [#12](https://github.com/ajepe/tryton/issues/12)
**Effort**: 2-4 person-weeks

JSON-RPC client with context injection, tagged codecs, caching, error handling. Used by all three clients.

**Deliverables**: `go/internal/client/`

#### Static File Serving
**Issue**: [#13](https://github.com/ajepe/tryton/issues/13)
**Effort**: 1-2 person-weeks

Serve web client build output from Go server with cache headers and gzip.

**Deliverables**: `go/internal/web/`

### Phase 5: Clients
**Issues**: [#9](https://github.com/ajepe/tryton/issues/9), [#10](https://github.com/ajepe/tryton/issues/10), [#11](https://github.com/ajepe/tryton/issues/11)
**Effort**: 20-36 person-weeks total

Build the user-facing clients for all three platforms.

#### Phase 5a: Web Client (TypeScript/Svelte)
**Issue**: [#9](https://github.com/ajepe/tryton/issues/9)
**Effort**: 8-12 person-weeks

**Stack**: Vite + Svelte 5 SPA with hash routing

- JSON-RPC client with context injection, tagged codecs, caching
- Login/session management, database selection
- Menu system, tree view, form view, relations
- Wizards, reports, export/import, attachments, notes
- Bus notifications, chat, i18n
- Responsive design (desktop + tablet)

**Deliverables**: `sao-ts/`

#### Phase 5b: Desktop Client (Wails)
**Issue**: [#10](https://github.com/ajepe/tryton/issues/10)
**Effort**: 4-6 person-weeks

**Stack**: Wails (Go host + webview), embeds web client

- Native file dialogs for import/export
- System tray with notification badges
- Native notifications (OS-level)
- Auto-update mechanism
- Offline mode with local cache (SQLite)
- Keyboard shortcuts (global + app-level)
- Multi-window support (tabs as windows)
- Build targets: Windows (.msi), macOS (.app), Linux (.AppImage/.deb)

**Deliverables**: `tryton-desktop/`

#### Phase 5c: Mobile Client (Flutter)
**Issue**: [#11](https://github.com/ajepe/tryton/issues/11)
**Effort**: 8-12 person-weeks

**Stack**: Flutter (Dart) for iOS and Android

- JSON-RPC client with context injection, tagged codecs, caching
- Login/session: biometric auth (Face ID/Touch ID), offline login
- Menu: server-driven, mobile-optimized navigation (drawer + bottom nav)
- Tree view: infinite scroll, pull-to-refresh, swipe actions
- Form view: mobile-optimized widgets, validation, on_change
- Relations: M2O search, O2M/M2M with modal editing
- Camera integration: barcode/QR scanning, image capture for attachments
- Push notifications (Firebase Cloud Messaging / APNs)
- Offline mode: local SQLite cache, sync on reconnect
- Build targets: iOS (.ipa), Android (.apk/.aab)

**Deliverables**: `tryton-mobile/`

### Phase 6: Cutover
**Issue**: [#8](https://github.com/ajepe/tryton/issues/8)
**Effort**: 2-4 person-weeks

Execute the migration from Python trytond to the Go server with zero data loss and minimal downtime.

**Deliverables**:
- `docs/go-rewrite/migration-guide.md`
- `docs/go-rewrite/rollback-procedure.md`
- `docs/go-rewrite/operations-runbook.md`
- `cutover/` — migration scripts and verification tools

## Effort Summary

| Phase | Effort (person-weeks) |
|-------|----------------------|
| 0 | 2-4 |
| 1 | 4-8 |
| 2 | 4-8 |
| 3 | 20-40 |
| 4 | 4-8 |
| Shared client lib | 2-4 |
| Static file serving | 1-2 |
| 5a (Web) | 8-12 |
| 5b (Desktop) | 4-6 |
| 5c (Mobile) | 8-12 |
| 6 | 2-4 |
| **Total** | **59-106** |

## Key Risks

1. **Module system semantics**: Python metaclasses, `_inherit`, dynamic registration — Go needs registry + codegen
2. **Schema migrations**: Must match Python's exact behavior for existing databases
3. **Business logic volume**: 194k LOC across 221 modules with cross-module inheritance
4. **XML DSL**: 2,223 XML files with inheritance, PYSON, access rules
5. **Reports**: ODF templates with Python expressions need a Go-compatible engine
6. **Ecosystem**: Third-party PyPI modules break; GPL-3 license applies

## License

GPL-3.0-or-later (derivative of Tryton, copyright B2CK/Cédric Krier and contributors)
