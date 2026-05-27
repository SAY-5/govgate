// Package api wires the checklist, scoring, and store packages into an HTTP
// service for submitting tools and querying the register.
package api

import (
	"context"
	"fmt"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
	"github.com/SAY-5/govgate/apps/register/internal/store"
)

// Service holds the loaded checklists and the register store.
type Service struct {
	store      store.Store
	checklists map[string]*checklist.Checklist
	def        string // default checklist name
}

// NewService builds a service. The default checklist name is used when a
// submission does not specify one.
func NewService(st store.Store, checklists map[string]*checklist.Checklist, def string) (*Service, error) {
	if _, ok := checklists[def]; !ok {
		return nil, fmt.Errorf("api: default checklist %q not loaded", def)
	}
	return &Service{store: st, checklists: checklists, def: def}, nil
}

// SubmitInput is a tool submission plus optional reviewer judgments and an
// explicit checklist choice.
type SubmitInput struct {
	Submission    scoring.Submission `json:"submission"`
	ChecklistName string             `json:"checklist_name,omitempty"`
	Judgments     scoring.Judgments  `json:"judgments,omitempty"`
}

// Submit scores a submission and persists the resulting register entry.
func (s *Service) Submit(ctx context.Context, in SubmitInput) (register.Entry, error) {
	name := in.ChecklistName
	if name == "" {
		name = s.def
	}
	c, ok := s.checklists[name]
	if !ok {
		return register.Entry{}, fmt.Errorf("api: unknown checklist %q", name)
	}
	assessment := scoring.Score(c, in.Submission, in.Judgments)
	entry := register.Entry{
		Submission:       in.Submission,
		ChecklistName:    c.Name,
		ChecklistVersion: c.Version,
		Assessment:       assessment,
		Status:           register.StatusPending,
	}
	return s.store.Insert(ctx, entry)
}

// Get returns one register entry.
func (s *Service) Get(ctx context.Context, id string) (register.Entry, error) {
	return s.store.Get(ctx, id)
}

// List returns a page of register entries.
func (s *Service) List(ctx context.Context, q register.Query) (register.Page, error) {
	return s.store.List(ctx, q)
}

// SetStatus updates an entry's review status and notes.
func (s *Service) SetStatus(ctx context.Context, id string, status register.Status, notes string) (register.Entry, error) {
	if !status.Valid() {
		return register.Entry{}, fmt.Errorf("api: invalid status %q", status)
	}
	return s.store.UpdateStatus(ctx, id, status, notes)
}

// Checklists returns the loaded checklist names.
func (s *Service) Checklists() map[string]*checklist.Checklist { return s.checklists }

// PublishChecklist registers a new checklist version and flags every entry
// assessed under an older version of the same checklist as stale. It returns
// the number of entries flagged.
func (s *Service) PublishChecklist(ctx context.Context, c *checklist.Checklist) (int, error) {
	s.checklists[c.Name] = c
	if c.Name == "" {
		return 0, fmt.Errorf("api: checklist name is required")
	}
	return s.store.MarkStale(ctx, c.Name, c.Version)
}

// Reassess re-scores an entry against the current version of its checklist,
// records the new assessment in history, and returns the updated entry plus a
// diff against the prior assessment.
func (s *Service) Reassess(ctx context.Context, id string) (register.Entry, register.AssessmentDiff, error) {
	entry, err := s.store.Get(ctx, id)
	if err != nil {
		return register.Entry{}, register.AssessmentDiff{}, err
	}
	c, ok := s.checklists[entry.ChecklistName]
	if !ok {
		return register.Entry{}, register.AssessmentDiff{}, fmt.Errorf("api: checklist %q no longer loaded", entry.ChecklistName)
	}

	prior := entry.Assessment
	newAssessment := scoring.Score(c, entry.Submission, entry.Assessment.Judgments())
	rec := register.AssessmentRecord{
		ChecklistName:    c.Name,
		ChecklistVersion: c.Version,
		Assessment:       newAssessment,
	}
	updated, err := s.store.Reassess(ctx, id, rec)
	if err != nil {
		return register.Entry{}, register.AssessmentDiff{}, err
	}
	diff := register.AssessmentDiff{
		FromVersion:  prior.ChecklistVersion,
		ToVersion:    c.Version,
		FromBand:     prior.OverallBand,
		ToBand:       newAssessment.OverallBand,
		BandRose:     newAssessment.OverallBand.Rank() > prior.OverallBand.Rank(),
		NewCriticals: diffCriticals(prior.Criticals, newAssessment.Criticals),
	}
	return updated, diff, nil
}

// History returns the assessment audit trail for an entry, oldest first.
func (s *Service) History(ctx context.Context, id string) ([]register.AssessmentRecord, error) {
	return s.store.History(ctx, id)
}

// diffCriticals returns critical ids present in next but not in prior.
func diffCriticals(prior, next []string) []string {
	seen := map[string]bool{}
	for _, c := range prior {
		seen[c] = true
	}
	var added []string
	for _, c := range next {
		if !seen[c] {
			added = append(added, c)
		}
	}
	return added
}
