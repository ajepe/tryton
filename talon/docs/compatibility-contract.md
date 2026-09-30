# Talon Compatibility Contract

## 1. Database Schema

Talon must produce and consume the same PostgreSQL schema as Python trytond.

### 1.1 Table Naming

- Model tables: `{module}_{model}` with dots replaced by underscores
  - Example: `party.party` → `party_party`
  - Example: `account.move.line` → `account_move_line`
- History tables: `{table}__history`
- Audit tables: `{table}__audit` (if auditing enabled)

### 1.2 Column Naming

- Field names: lowercase with underscores
- Special columns:
  - `id` — primary key, serial
  - `create_uid` — FK to `res_user.id`
  - `create_date` — timestamp with time zone
  - `write_uid` — FK to `res_user.id`
  - `write_date` — timestamp with time zone
  - `_timestamp` — JSONB, per-field timestamps for optimistic concurrency

### 1.3 Field Type Mapping

| Tryton Field Type | PostgreSQL Type |
|-------------------|-----------------|
| `char` | `varchar` (with optional max length) |
| `text` | `text` |
| `integer` | `integer` |
| `float` | `double precision` |
| `numeric` | `numeric` |
| `boolean` | `boolean` |
| `date` | `date` |
| `datetime` | `timestamp with time zone` |
| `time` | `time` |
| `timedelta` | `interval` |
| `binary` | `bytea` |
| `selection` | `varchar` |
| `reference` | `varchar` |
| `many2one` | `integer` (FK) |
| `one2many` | (separate table) |
| `many2many` | (join table) |
| `dict` | `jsonb` |

### 1.4 Constraints

- Primary keys: `id`
- Foreign keys: `create_uid`, `write_uid` → `res_user.id`
- Unique constraints: as defined by field `unique=True`
- Check constraints: as defined by field domains
- Indexes: `btree` on FK columns, `gin` on `jsonb` columns

### 1.5 Sequences

- Table sequences: `{table}_id_seq`
- Named sequences: `ir_sequence` model with `ir_sequence_type`

## 2. JSON-RPC Protocol

### 2.1 Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/{db}/rpc/` | JSON-RPC dispatch for `model.*`, `wizard.*`, `report.*`, `common.db.login`, `common.db.logout`, `system.*` |
| POST | `/rpc/` | Unauthenticated RPC: `common.server.version`, `common.db.list`, `common.authentication.services` |
| POST | `/{db}/session/login` | Login, sets `tryton_session` cookie |
| POST | `/{db}/session/logout` | Logout |
| POST | `/{db}/bus` | Long-poll bus |
| GET | `/{db}/r/<path>` | Static files, reports, avatars |

### 2.1.1 Tryton 8 search API

Tryton 8 changed the search argument order. Talon accepts both forms but emits
the 8.x form from its own clients:

- `model.<m>.search(domain, offset, limit, order)` — `order` is a list of
  `[field, "ASC"|"DESC"]` pairs (a plain string is accepted for compatibility).
- `model.<m>.search_read(domain, offset, limit, order, fields)`.
- `model.<m>.search_count(domain, offset, limit)`.

All three append the context dict as the last parameter at the RPC layer.
`search_read` always includes `id` in each returned row when requested fields
omit it.

### 2.2 Request Envelope

```json
{
  "id": 1,
  "method": "model.party.party.read",
  "params": [[1, 2], ["name", "code"], {"language": "en"}]
}
```

- `id`: integer, echoed in response
- `method`: string, dot-separated namespace
- `params`: array, last element is always context dict

### 2.3 Response Envelope

Success:
```json
{
  "id": 1,
  "result": [...]
}
```

Error:
```json
{
  "id": 1,
  "error": ["UserError", ["Message", "Description", "Domain"]]
}
```

### 2.4 Tagged Value Codecs

| Type | JSON Representation |
|------|-------------------|
| `datetime` | `{"__class__": "datetime", "year": 2024, "month": 1, "day": 15, "hour": 10, "minute": 30, "second": 0}` |
| `date` | `{"__class__": "date", "year": 2024, "month": 1, "day": 15}` |
| `time` | `{"__class__": "time", "hour": 10, "minute": 30, "second": 0}` |
| `timedelta` | `{"__class__": "timedelta", "seconds": 3600}` |
| `Decimal` | `{"__class__": "Decimal", "decimal": "123.45"}` |
| `bytes` | `{"__class__": "bytes", "base64": "aGVsbG8="}` |

### 2.5 Context

The last parameter of every RPC method is a context dict:

```json
{
  "language": "en",
  "company": 1,
  "_timestamp": {"party.party,1": "2024-01-15 10:30:00"},
  "_user": 1
}
```

- `_timestamp`: consumed by server for optimistic concurrency
- `_user`: current user ID
- All other keys passed to model methods

### 2.6 Caching

Methods may return `X-Tryton-Cache: <seconds>` header. Clients cache by method+params.

### 2.7 Error Types

| Error | HTTP Code | JSON-RPC Error |
|-------|-----------|----------------|
| `UserError` | 200 | `["UserError", [message, description, domain]]` |
| `UserWarning` | 409 | `["UserWarning", [name, message, description]]` |
| `ConcurrencyException` | 423 | `["ConcurrencyException", [message]]` |
| `LoginException` | 403 | `["LoginException", [name, message, type]]` |
| `RateLimitException` | 429 | `["RateLimitException", [message]]` |

## 3. Session Management

### 3.1 Login

`POST /{db}/session/login` with JSON body:
```json
{"id": 0, "method": "common.db.login", "params": ["admin", {"password": "secret"}, "en"]}
```

Response: `[user_id, bus_url_host]`

Sets cookie: `tryton_session=login:userid:token; Secure; HttpOnly; SameSite=Strict; Path=/{db}`

### 3.2 Token Auth

`Authorization: session <base64("login:userid:token")>`

### 3.3 Session Renewal

On 401, client re-authenticates and retries the original request.

### 3.4 Password Storage

Tryton 8 stores credential hashes in `res_user.password_hash` using Argon2id
(`$argon2id$v=19$m=65536,t=3,p=4$<salt>$<digest>`, 16-byte salt, 32-byte
digest). Talon verifies Argon2id and legacy bcrypt (`$2a$`/`$2b$`) hashes and
writes Argon2id hashes, so hashes are interchangeable between servers.
The legacy `res_user.password` column is read as a fallback for older
databases.

### 3.5 Value Preprocessing

Values are normalized server-side on `create` and `write` (Tryton
`preprocess_values`), e.g. `country.country.code` is upper-cased. Clients must
not rely on `on_change` for normalization.

## 4. Module System

### 4.1 Module Definition

Each module has a `tryton.cfg`:
```ini
[tryton]
version=8.1.0
depends:
    country
    ir
    res
xml:
    party.xml
    category.xml

[register]
model:
    party.Party
    party.PartyLang
wizard:
    party.Replace
```

### 4.2 Module Registration

Models register via `Pool.register` at module load. In Talon, this becomes a Go init-time registry:

```go
func init() {
    metadata.RegisterModel("party.Party", Party{})
}
```

### 4.3 Module Upgrade

`ir.module.module.upgrade(modules)` runs versioned upgrade functions. In Talon:

```go
func Upgrade_8_0_0(ctx context.Context) error { ... }
```

## 5. XML Data Format

### 5.1 View Definitions

```xml
<record model="ir.ui.view" id="party_view_form">
    <field name="model">party.party</field>
    <field name="type">form</field>
    <field name="arch" type="xml">
        <form>
            <field name="name"/>
            <field name="code"/>
            <notebook>
                <page string="General">
                    <field name="addresses"/>
                </page>
            </notebook>
        </form>
    </field>
</record>
```

### 5.2 Inheritance

```xml
<record model="ir.ui.view" id="party_view_form_inherit" inherit="party_view_form">
    <field name="arch" type="xml">
        <xpath expr="//field[@name='name']" position="after">
            <field name="category"/>
        </xpath>
    </field>
</record>
```

### 5.3 Actions

```xml
<record model="ir.action.act_window" id="act_party">
    <field name="name">Parties</field>
    <field name="res_model">party.party</field>
    <field name="view_mode">tree,form</field>
</record>
```

## 6. PYSON

PYSON is a safe expression language for domains, contexts, and states.

### 6.1 Syntax

```
domain := [field, operator, value]
       := [domain, 'AND', domain]
       := [domain, 'OR', domain]
       := ['NOT', domain]
       := [field, 'in', [values]]
       := [field, 'not in', [values]]
       := [field, 'like', 'pattern']
       := [field, 'ilike', 'pattern']
       := [field, 'child_of', value]
       := [field, 'parent_of', value]
```

### 6.2 Operators

`=`, `!=`, `<`, `<=`, `>`, `>=`, `in`, `not in`, `like`, `not like`, `ilike`, `not ilike`, `child_of`, `parent_of`

### 6.3 Functions

`Eval`, `Context`, `ActiveID`, `ActiveIds`, `CurrentCompany`, `CurrentLanguage`

## 7. Bus / Notifications

### 7.1 Long-Poll

`POST /{db}/bus` with body:
```json
{"channels": ["notification:1"], "last_message": "uuid-or-null"}
```

Response:
```json
{"channel": "notification:1", "message": {"type": "notification", "title": "...", "body": "..."}}
```

### 7.2 Message Types

- `notification` — browser popup
- `user-notification` — menu badge update
- `message` — chat message

## 8. Reports

### 8.1 Execution

`report.<name>.execute(ids, data, context)` returns `[extension, content, _, filename]`

### 8.2 Templates

ODF/ODS templates with Python-compatible expressions. Talon uses a Go template engine with PYSON-compatible syntax.

## 9. Conformance Testing

The conformance harness runs identical requests against Python trytond and Talon, comparing responses with tolerance for:
- Field ordering
- Timestamp precision
- ID assignment order
