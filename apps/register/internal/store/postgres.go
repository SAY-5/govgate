package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
)

// Postgres is a pgx-backed Store.
type Postgres struct {
	pool *pgxpool.Pool
}

// Open connects to Postgres and applies the schema.
func Open(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	p := &Postgres{pool: pool}
	if err := p.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS register_entries (
    id                 UUID PRIMARY KEY,
    vendor             TEXT NOT NULL,
    tool_name          TEXT NOT NULL,
    checklist_name     TEXT NOT NULL,
    checklist_version  TEXT NOT NULL,
    overall_band       TEXT NOT NULL,
    status             TEXT NOT NULL,
    stale              BOOLEAN NOT NULL DEFAULT FALSE,
    reviewer_notes     TEXT NOT NULL DEFAULT '',
    submission         JSONB NOT NULL,
    assessment         JSONB NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_register_status ON register_entries (status);
CREATE INDEX IF NOT EXISTS idx_register_band   ON register_entries (overall_band);
CREATE INDEX IF NOT EXISTS idx_register_vendor ON register_entries (vendor);
CREATE INDEX IF NOT EXISTS idx_register_created ON register_entries (created_at DESC, id DESC);
`

func (p *Postgres) migrate(ctx context.Context) error {
	if _, err := p.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	return nil
}

// Close releases the pool.
func (p *Postgres) Close() { p.pool.Close() }

// Insert persists a new entry.
func (p *Postgres) Insert(ctx context.Context, e register.Entry) (register.Entry, error) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	subJSON, err := json.Marshal(e.Submission)
	if err != nil {
		return register.Entry{}, fmt.Errorf("store: marshal submission: %w", err)
	}
	assJSON, err := json.Marshal(e.Assessment)
	if err != nil {
		return register.Entry{}, fmt.Errorf("store: marshal assessment: %w", err)
	}
	const q = `
INSERT INTO register_entries
  (id, vendor, tool_name, checklist_name, checklist_version, overall_band, status, stale, reviewer_notes, submission, assessment)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING created_at, updated_at`
	row := p.pool.QueryRow(ctx, q,
		e.ID, e.Submission.Vendor, e.Submission.Name, e.ChecklistName, e.ChecklistVersion,
		string(e.Assessment.OverallBand), string(e.Status), e.Stale, e.ReviewerNotes, subJSON, assJSON,
	)
	if err := row.Scan(&e.CreatedAt, &e.UpdatedAt); err != nil {
		return register.Entry{}, fmt.Errorf("store: insert: %w", err)
	}
	return e, nil
}

// Get fetches one entry by id.
func (p *Postgres) Get(ctx context.Context, id string) (register.Entry, error) {
	const q = `
SELECT id, checklist_name, checklist_version, status, stale, reviewer_notes,
       submission, assessment, created_at, updated_at
FROM register_entries WHERE id = $1`
	return scanEntry(p.pool.QueryRow(ctx, q, id), id)
}

// List returns a page of entries newest-first, with keyset pagination on
// (created_at, id).
func (p *Postgres) List(ctx context.Context, query register.Query) (register.Page, error) {
	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var conds []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if query.Status != "" {
		add("status = $%d", string(query.Status))
	}
	if query.Band != "" {
		add("overall_band = $%d", string(query.Band))
	}
	if query.Vendor != "" {
		add("vendor = $%d", query.Vendor)
	}
	if query.Cursor != "" {
		// page after the cursor entry's (created_at, id)
		conds = append(conds, fmt.Sprintf(
			"(created_at, id) < (SELECT created_at, id FROM register_entries WHERE id = $%d)",
			len(args)+1))
		args = append(args, query.Cursor)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, limit+1)
	q := fmt.Sprintf(`
SELECT id, checklist_name, checklist_version, status, stale, reviewer_notes,
       submission, assessment, created_at, updated_at
FROM register_entries %s
ORDER BY created_at DESC, id DESC
LIMIT $%d`, where, len(args))

	rows, err := p.pool.Query(ctx, q, args...)
	if err != nil {
		return register.Page{}, fmt.Errorf("store: list: %w", err)
	}
	defer rows.Close()

	var page register.Page
	for rows.Next() {
		e, err := scanEntry(rows, "")
		if err != nil {
			return register.Page{}, err
		}
		page.Entries = append(page.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return register.Page{}, fmt.Errorf("store: list scan: %w", err)
	}
	if len(page.Entries) > limit {
		page.NextCursor = page.Entries[limit-1].ID
		page.Entries = page.Entries[:limit]
	}
	return page, nil
}

// UpdateStatus changes the status and reviewer notes.
func (p *Postgres) UpdateStatus(ctx context.Context, id string, status register.Status, notes string) (register.Entry, error) {
	const q = `
UPDATE register_entries SET status = $2, reviewer_notes = $3, updated_at = now()
WHERE id = $1
RETURNING id, checklist_name, checklist_version, status, stale, reviewer_notes,
          submission, assessment, created_at, updated_at`
	e, err := scanEntry(p.pool.QueryRow(ctx, q, id, string(status), notes), id)
	if err != nil {
		return register.Entry{}, err
	}
	return e, nil
}

// rowScanner abstracts pgx.Row and pgx.Rows for scanEntry.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(row rowScanner, id string) (register.Entry, error) {
	var (
		e       register.Entry
		statusS string
		subJSON []byte
		assJSON []byte
		created time.Time
		updated time.Time
	)
	err := row.Scan(&e.ID, &e.ChecklistName, &e.ChecklistVersion, &statusS, &e.Stale,
		&e.ReviewerNotes, &subJSON, &assJSON, &created, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return register.Entry{}, ErrNotFound{ID: id}
	}
	if err != nil {
		return register.Entry{}, fmt.Errorf("store: scan: %w", err)
	}
	var sub scoring.Submission
	if err := json.Unmarshal(subJSON, &sub); err != nil {
		return register.Entry{}, fmt.Errorf("store: unmarshal submission: %w", err)
	}
	var ass scoring.Assessment
	if err := json.Unmarshal(assJSON, &ass); err != nil {
		return register.Entry{}, fmt.Errorf("store: unmarshal assessment: %w", err)
	}
	e.Submission = sub
	e.Assessment = ass
	e.Status = register.Status(statusS)
	e.CreatedAt = created
	e.UpdatedAt = updated
	return e, nil
}
