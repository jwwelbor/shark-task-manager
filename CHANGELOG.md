# Changelog

All notable changes to Shark Task Manager are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased] — E07-F42

### Added

#### `--size` flag on all 6 entity `create` commands

All six entity creation commands now accept an optional `--size` flag:

```
shark epic create    ... --size <value>
shark feature create ... --size <value>
shark task create    ... --size <value>
shark bug create     ... --size <value>
shark change create  ... --size <value>
shark idea create    ... --size <value>
```

`<value>` accepts either the **t-shirt label** form or the **Fibonacci numeric** form:

| Label | Numeric |
|-------|---------|
| `XS`  | `1`     |
| `S`   | `2`     |
| `M`   | `3`     |
| `L`   | `5`     |
| `XL`  | `8`     |
| `XXL` | `13`    |

Both forms are equivalent on input. The canonical stored value is always numeric.
The flag is optional; absence leaves `size` as `NULL`.
Invalid values (e.g., `--size 4`, `--size XXXL`) exit non-zero and print the
allowed set to stderr.

#### `--size` flag on all 6 entity `update` commands

All six entity update commands also accept `--size`:

```
shark epic update    <key> --size <value>
shark feature update <key> --size <value>
shark task update    <key> --size <value>
shark bug update     <key> --size <value>
shark change update  <key> --size <value>
shark idea update    <key> --size <value>
```

Sentinel behavior on `update`:

- **Flag absent** — no change to the stored size.
- **`--size <valid-value>`** — update to that size.
- **`--size clear`** (literal string) — set the size back to `NULL`.

#### Size in JSON output

All six entity types include two new fields in their JSON output:

```json
{
  "size": 5,
  "size_label": "L"
}
```

- `size` — canonical integer (`null | 1 | 2 | 3 | 5 | 8 | 13`).
- `size_label` — derived t-shirt label (`null | "XS" | "S" | "M" | "L" | "XL" | "XXL"`).

When size is `NULL`, both fields are omitted from JSON output (treated as absent
by the `omitempty` serialization tag). The `--field size` extractor exits with
code **4** (field not found / null sentinel) for unsized entities, matching the
existing convention for NULL fields.

`size_label` is currently emitted only by `get` / `status` commands and the
`--field size_label` extractor; create-command JSON returns the canonical
numeric `size` only.

#### Human-readable output

Non-JSON output renders size as `<label> (<numeric>)` — e.g., `L (5)` — or as
`—` when size is unset.

#### Template variables `{{ .size }}` and `{{ .size_label }}`

Entity file templates now have access to two new placeholders:

- `{{ .size }}` — numeric form (e.g., `5`), or empty string when unset.
- `{{ .size_label }}` — label form (e.g., `L`), or empty string when unset.

These are independent from `complexity_tier` (see deprecation notice below).

#### Database migration (schema version 15)

A `size INTEGER NULL` column has been added to six tables:
`epics`, `features`, `tasks`, `bugs`, `change_cards`, `ideas`.
The migration is idempotent and runs automatically on first use after upgrade.
Existing rows are unaffected; their `size` values default to `NULL`.

---

### Deprecated

#### `complexity_tier` metadata field

The `complexity_tier` field stored in entity `Metadata` maps is **deprecated**
as of this release. It remains fully functional for one release cycle to allow
gradual migration:

- Templates using `{{ .complexity_tier }}` continue to work unchanged.
- The `complexity_tier` value is **not** automatically migrated to `size`.
  To backfill, re-run `shark task update <key> --size <value>` (or the equivalent
  command for your entity type) for each entity you want to size explicitly.
  A one-shot batch migration command (`shark admin migrate complexity-to-size`)
  may be provided in a future release.

**Planned removal:** `complexity_tier` extraction from `Metadata` will be removed
in a future release. Users relying on `complexity_tier` in templates should
migrate to `{{ .size_label }}` and `{{ .size }}` before that release.

If you currently set `complexity_tier` via the `Metadata` map, start using
`--size` on `create`/`update` commands instead.

---

### Notes

- The HTTP API (`cmd/server/`) does not yet expose `--size` parity; this is
  deferred to a follow-up (see spec OOS-7).
- Size-based rollups, analytics, and workflow gates (e.g., blocking a transition
  if size > M) are not included in this release (see spec OOS-2, OOS-4).
- The `tech_debt` entity type is excluded from this feature because it has no
  user-facing CLI surface (see spec OOS-3).

---

*Prior releases did not use a CHANGELOG. This file was created as part of E07-F42.*
