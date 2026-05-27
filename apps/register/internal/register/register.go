// Package register holds the domain model for the review register: the record
// of every tool that has been submitted and assessed, plus the query surface
// over those records.
package register

import (
	"time"

	"github.com/SAY-5/govgate/apps/register/internal/scoring"
)

// Status is the review state of a tool in the register.
type Status string

const (
	StatusPending       Status = "pending"
	StatusApproved      Status = "approved"
	StatusRejected      Status = "rejected"
	StatusNeedsInfo     Status = "needs-info"
	StatusFullyApproved Status = "fully-approved"
)

// Valid reports whether s is a recognized status.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusApproved, StatusRejected, StatusNeedsInfo, StatusFullyApproved:
		return true
	}
	return false
}

// Entry is a single tool's record in the register, including its most recent
// assessment.
type Entry struct {
	ID               string             `json:"id"`
	Submission       scoring.Submission `json:"submission"`
	ChecklistName    string             `json:"checklist_name"`
	ChecklistVersion string             `json:"checklist_version"`
	Assessment       scoring.Assessment `json:"assessment"`
	Status           Status             `json:"status"`
	ReviewerNotes    string             `json:"reviewer_notes"`
	Stale            bool               `json:"stale_assessment"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// Query filters the register listing. Empty fields are not constrained.
type Query struct {
	Status Status
	Band   scoring.Band
	Vendor string
	Cursor string // opaque; the id to page after
	Limit  int
}

// Page is a slice of register entries plus the cursor for the next page.
type Page struct {
	Entries    []Entry `json:"entries"`
	NextCursor string  `json:"next_cursor,omitempty"`
}
