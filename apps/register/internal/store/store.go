// Package store provides the Postgres-backed persistence for the register.
package store

import (
	"context"

	"github.com/SAY-5/govgate/apps/register/internal/register"
)

// Store is the persistence surface for register entries.
type Store interface {
	// Insert persists a new entry and returns it with server-set timestamps.
	Insert(ctx context.Context, e register.Entry) (register.Entry, error)
	// Get fetches a single entry by id.
	Get(ctx context.Context, id string) (register.Entry, error)
	// List returns a page of entries matching the query, newest first.
	List(ctx context.Context, q register.Query) (register.Page, error)
	// UpdateStatus changes the status and reviewer notes of an entry.
	UpdateStatus(ctx context.Context, id string, status register.Status, notes string) (register.Entry, error)
	// MarkStale flags every entry whose checklist version differs from current
	// for the given checklist name, returning the count flagged.
	MarkStale(ctx context.Context, checklistName, currentVersion string) (int, error)
	// Reassess replaces an entry's live assessment, appends the new assessment
	// to its history, and clears the stale flag.
	Reassess(ctx context.Context, id string, a register.AssessmentRecord) (register.Entry, error)
	// History returns every assessment an entry has received, oldest first.
	History(ctx context.Context, id string) ([]register.AssessmentRecord, error)
	// Close releases resources.
	Close()
}

// ErrNotFound is returned when an entry does not exist.
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return "register entry not found: " + e.ID }
