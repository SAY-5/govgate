package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/SAY-5/govgate/apps/register/internal/register"
)

// AddConditions attaches conditions to an entry, assigning ids and timestamps.
func (p *Postgres) AddConditions(ctx context.Context, entryID string, conds []register.Condition) ([]register.Condition, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const q = `
INSERT INTO approval_conditions (id, entry_id, description, due_at)
VALUES ($1,$2,$3,$4)
RETURNING created_at`
	out := make([]register.Condition, 0, len(conds))
	for _, c := range conds {
		c.ID = uuid.NewString()
		c.EntryID = entryID
		if err := tx.QueryRow(ctx, q, c.ID, entryID, c.Description, c.DueAt).Scan(&c.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: add condition: %w", err)
		}
		out = append(out, c)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}
	return out, nil
}

// Conditions returns all conditions attached to an entry, oldest first.
func (p *Postgres) Conditions(ctx context.Context, entryID string) ([]register.Condition, error) {
	const q = `
SELECT id, entry_id, description, due_at, satisfied, evidence, satisfied_at, created_at
FROM approval_conditions WHERE entry_id = $1 ORDER BY created_at ASC, id ASC`
	rows, err := p.pool.Query(ctx, q, entryID)
	if err != nil {
		return nil, fmt.Errorf("store: conditions: %w", err)
	}
	defer rows.Close()
	var out []register.Condition
	for rows.Next() {
		c, err := scanCondition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SatisfyCondition marks a condition satisfied with evidence at the given time.
func (p *Postgres) SatisfyCondition(ctx context.Context, entryID, condID, evidence string, at time.Time) (register.Condition, error) {
	const q = `
UPDATE approval_conditions SET satisfied = TRUE, evidence = $3, satisfied_at = $4
WHERE id = $1 AND entry_id = $2
RETURNING id, entry_id, description, due_at, satisfied, evidence, satisfied_at, created_at`
	row := p.pool.QueryRow(ctx, q, condID, entryID, evidence, at)
	c, err := scanCondition(row)
	if err != nil {
		return register.Condition{}, err
	}
	return c, nil
}

// Overdue returns entries with unsatisfied conditions past due as of now.
func (p *Postgres) Overdue(ctx context.Context, now time.Time) ([]register.OverdueEntry, error) {
	const idsQ = `
SELECT DISTINCT entry_id FROM approval_conditions
WHERE satisfied = FALSE AND due_at < $1`
	rows, err := p.pool.Query(ctx, idsQ, now)
	if err != nil {
		return nil, fmt.Errorf("store: overdue ids: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("store: overdue scan: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []register.OverdueEntry
	for _, id := range ids {
		entry, err := p.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		conds, err := p.Conditions(ctx, id)
		if err != nil {
			return nil, err
		}
		var overdue []register.Condition
		for _, c := range conds {
			if c.Overdue(now) {
				overdue = append(overdue, c)
			}
		}
		if len(overdue) > 0 {
			out = append(out, register.OverdueEntry{Entry: entry, Conditions: overdue})
		}
	}
	return out, nil
}

func scanCondition(row rowScanner) (register.Condition, error) {
	var (
		c           register.Condition
		satisfiedAt *time.Time
	)
	if err := row.Scan(&c.ID, &c.EntryID, &c.Description, &c.DueAt, &c.Satisfied,
		&c.Evidence, &satisfiedAt, &c.CreatedAt); err != nil {
		return register.Condition{}, fmt.Errorf("store: scan condition: %w", err)
	}
	c.SatisfiedAt = satisfiedAt
	return c, nil
}
