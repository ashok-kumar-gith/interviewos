# Migrations

Schema evolves **only** through versioned migrations applied by `golang-migrate`
(run via `cmd/migrate`). This includes seeded/curated curriculum and company
content — it ships as migrations, never ad-hoc SQL.

## File convention

Each migration is a pair of ordered SQL files using the `golang-migrate`
naming scheme:

```
NNNNNN_short_name.up.sql     # forward migration
NNNNNN_short_name.down.sql   # reverse migration
```

- `NNNNNN` is a zero-padded, strictly increasing sequence number (e.g.
  `000001`). Numbers must be unique and ordered.
- `short_name` is a snake_case description (e.g. `create_users`).
- Every `.up.sql` MUST have a matching `.down.sql` that cleanly reverses it.
- Migrations are immutable once merged; corrections go in a new migration.

## Examples

```
000001_create_users.up.sql
000001_create_users.down.sql
000002_create_content_spine.up.sql
000002_create_content_spine.down.sql
```

## Running

```
# apply all pending migrations
go run ./cmd/migrate

# (golang-migrate up/down/force wiring lands with the first migration)
```

The readiness probe (`/readyz`) gates rollout on migrations being applied.
