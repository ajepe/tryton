# Talon Rewrite Status

Last updated: 2026-09-30

## Phase 0 — Contract + conformance harness — **In progress**

- Compatibility contract: `talon/docs/compatibility-contract.md`
- Differential runner: `talon/conformance` (fixtures, per-case setup, ignored
  fields, list sorting, number normalization, session authentication)
- Cases: `talon/conformance/cases` (auth/common/model), fixtures in
  `talon/conformance/fixtures`
- CLI: `talon/cmd/conformance` (login, fixtures, report, non-zero exit on
  mismatches)
- **Verified against live Python trytond 8.1 on a shared database**: 7 of 8
  curated cases match byte-for-byte after normalization
  (`common.authentication.services`, `common.db.list`,
  `common.db.logout`, `country.country.on_change`, `product.uom.on_change`,
  `party.party.search_read`, `party.party.search_count`). The only mismatch is
  `common.server.version`, which reports different product versions by design
  and is skipped in the default case set.
- The differential run found and fixed real divergences: Tryton 8
  `search_read(domain, offset, limit, order, fields)` argument order, order as
  `[field, direction]` pairs, root `/rpc/` routing for common methods,
  Argon2id `password_hash` authentication, and write-time `preprocess_values`.
- Not done: running against Python `trytond` in CI (needs a pinned environment;
  the local procedure is documented here and in the migration guide).

## Phase 1 — Go core engine — **Done (verified against PostgreSQL)**

- Metadata registry with all core models (`talon/internal/metadata`,
  `talon/internal/modules/*`)
- ORM with batched record hydration, create/write/delete/load, x2m copy,
  quoted identifiers (`talon/internal/orm`)
- Schema sync: ordered creation, standard columns, m2m relation tables,
  indexes, additive `ALTER TABLE` (`talon/internal/schema`)
- PostgreSQL backend on pgx (`talon/internal/backend/postgresql`)
- Integration tests run against a real PostgreSQL server
  (`talon/internal/integration`, `talon/internal/testsupport`).

## Phase 2 — Request layer — **Done (core paths)**

- JSON-RPC server with placeholder routing for dotted model names
  (`model.party.party.read`), tagged codecs, caches, error mapping
  (`talon/internal/rpc`)
- Sessions: bcrypt authentication against `res_user`, rate limiting, cookies,
  `Authorization: session <base64>`, HTTP login/logout endpoints
  (`talon/internal/session`)
- Security: `ir_model_access` / `ir_model_field_access` checks and `ir.rule` /
  `ir.rule.group` record rules (global and per-group, PYSON domains) integrated
  into model RPC search/read/write/delete (`talon/internal/security`,
  `talon/internal/pyson`)
- i18n: `ir_translation` lookup with language fallback, `ir_lang` metadata,
  number/date formatting, and PO import (`talon/internal/i18n`)
- XML data loader: records, nested records, `ref`, `eval`, `ir_model_data`
  upserts (`talon/internal/xml`)
- Actions: act_window/wizard/report/url resolution, view lookup, keywords
  (`talon/internal/action`)
- Views: XML arch DOM, xpath inheritance (replace/inside/after/before/
  attributes), `fields_view_get` returning merged arch
  (`talon/internal/xml`, `talon/internal/rpc`)
- PYSON decoder/encoder (`talon/internal/pyson`) — evaluator covers Eval,
  Context, ActiveID(s), CurrentCompany/Language.
- Not done: i18n of view arch strings, full Python-expression evaluation
  beyond PYSON literals/calls.

## Phase 3 — Module ports — **Core subset done**

Ported with business logic and tests: `ir`, `res`, `country`, `currency`,
`company`, `party`, `product`, `account`, `stock`, `sale`, `purchase`.
Handlers are reachable via standard Tryton method names
(`model.<model>.<action>`, `on_change`, `on_change_with`, `pre_validate`) and
run on create/write. See `talon/internal/modules`.

Not done: full field sets per module, taxes, payment terms, invoice/shipment
generation, stock valuation, schedules, and the remaining 212 modules.

## Phase 4 — Platform services — **Done (core paths)**

- Bus: in-memory hub, long-poll endpoint, PostgreSQL LISTEN/NOTIFY bridge
  (`talon/internal/bus`)
- Wizard engine + RPC (`talon/internal/wizard`)
- Reports: registry, FuncReport, CSV report, ODF/FODT template rendering
  (`${object.field}` substitution, zip rewrite) and `report.<name>.execute`
  with a `-report-dir` loader (`talon/internal/report`)
- Cron: next-call computation, scheduler loop, SQL store
  (`talon/internal/cron`)
- Filestore: SHA-256 content-addressed storage + attachment service
  (`talon/internal/filestore`)
- Email: MIME builder, SMTP sender, memory sender (`talon/internal/email`)
- Web: static serving with gzip, immutable caching and SPA fallback
  (`talon/internal/web`)

Not done: ODF template rendering, wizard form/dialog UX details, worker pool.

## Phase 5 — Clients

- **5a Web** (`sao-ts`): Vite + Svelte 5 SPA. Login/database select, menu and
  model browser, list view with search/pagination, form view with on_change,
  notifications drawer, bus long-poll, tagged codecs. Builds with
  `npm run build`.
- **5b Desktop** (`tryton-desktop`): Wails project wrapping the web build,
  native file dialogs and notifications. Scaffold; requires Wails toolchain.
- **5c Mobile** (`tryton-mobile`): Flutter project with RPC client, codec,
  bus client, login/menu/tree/form screens. Scaffold; requires Flutter
  toolchain.

## Phase 6 — Cutover — **Docs and scripts done**

- `docs/go-rewrite/migration-guide.md`
- `docs/go-rewrite/rollback-procedure.md`
- `docs/go-rewrite/operations-runbook.md`
- `cutover/snapshot.sh`, `cutover/verify.sh`, `cutover/verify_rpc.sh`

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` pass.
- Integration tests cover: schema creation, CRUD over HTTP JSON-RPC, sessions,
  access rules, translations, XML loading, actions, attachments, bus
  notifications, module on_change/buttons.
- End-to-end smoke test with the real binary and PostgreSQL passes
  (`cutover/verify_rpc.sh`).
- Web client production build passes (`sao-ts/dist`).

## Known gaps

1. Module coverage is a core subset; the remaining modules still need ports.
2. Reports support placeholder substitution but not relatorio loops/expressions.
3. Desktop/mobile clients are scaffolds and need their toolchains plus
   platform-specific features (biometrics, push, offline cache).
4. Schema sync is additive only; destructive migrations are not automated.
5. View arch translation (inline `string`/`help` attributes) is not yet
   resolved through `ir_translation`.
