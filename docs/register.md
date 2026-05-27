# Register API

The register is a Postgres-backed record of every tool that has been submitted
and assessed. The Go service exposes it over a small REST API.

## Endpoints

### `POST /v1/submissions`

Submit a tool. The body carries the submission, an optional `checklist_name`
(defaults to `default`), and optional `judgments` for non-auto requirements.

```json
{
  "submission": {
    "name": "Acme Chat",
    "vendor": "Acme AI",
    "description": "...",
    "intended_use": "...",
    "structured_fields": { "encryption_in_transit": true }
  },
  "checklist_name": "default",
  "judgments": { "ho.human_in_loop": "pass" }
}
```

The service scores the submission and persists a register entry with status
`pending`. Returns `201` with the created entry, including its assessment.

### `GET /v1/register/{id}`

Fetch a single entry by id. Returns `404` if it does not exist.

### `GET /v1/register`

List entries, newest first. Query parameters:

| Param | Meaning |
|-------|---------|
| `status` | filter by review status |
| `band` | filter by overall risk band |
| `vendor` | filter by vendor (exact) |
| `cursor` | opaque keyset cursor for the next page |
| `limit` | page size (default 50, max 200) |

Returns `{ "entries": [...], "next_cursor": "..." }`. Pagination is keyset on
`(created_at, id)`, so it is stable under concurrent inserts.

### `POST /v1/register/{id}/status`

Update review status and reviewer notes.

```json
{ "status": "approved", "reviewer_notes": "DPA on file" }
```

Valid statuses: `pending`, `approved`, `rejected`, `needs-info`,
`fully-approved`.

### `POST /v1/register/{id}/reassess`

Re-score an entry against the current version of its checklist. Returns the
updated entry and a diff against the prior assessment. See
[versioning.md](versioning.md).

### `GET /v1/register/{id}/history`

Return every assessment the entry has received, oldest first (the audit trail).

## Storage

Entries are stored in `register_entries`. The submission and assessment are kept
as `JSONB`; vendor, tool name, checklist coordinates, overall band, and status
are projected into indexed columns for fast filtering. The schema is applied
idempotently at startup.

## Testing

Integration tests use [testcontainers](https://golang.testcontainers.org/) to
run a throwaway Postgres, so they are hermetic. Build them with the
`integration` tag: `go test -tags=integration ./...`. In CI, set
`TESTCONTAINERS_RYUK_DISABLED=true`.
