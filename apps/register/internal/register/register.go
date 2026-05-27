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

// AssessmentRecord is one assessment a tool received at a point in time. The
// sequence of records for an entry is the audit trail of how its risk changed
// as checklist policy evolved.
type AssessmentRecord struct {
	ID               string             `json:"id"`
	ChecklistName    string             `json:"checklist_name"`
	ChecklistVersion string             `json:"checklist_version"`
	Assessment       scoring.Assessment `json:"assessment"`
	CreatedAt        time.Time          `json:"created_at"`
}

// AssessmentDiff summarizes how the overall band changed between two
// assessments of the same tool.
type AssessmentDiff struct {
	FromVersion  string       `json:"from_version"`
	ToVersion    string       `json:"to_version"`
	FromBand     scoring.Band `json:"from_band"`
	ToBand       scoring.Band `json:"to_band"`
	BandRose     bool         `json:"band_rose"`
	NewCriticals []string     `json:"new_criticals"`
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
