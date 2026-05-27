package api

import (
	"context"
	"fmt"

	"github.com/SAY-5/govgate/apps/register/internal/register"
)

// ConditionInput describes one condition to attach on conditional approval.
type ConditionInput struct {
	Description string `json:"description"`
	DueInDays   int    `json:"due_in_days"`
}

// ApproveWithConditions sets the entry to approved and attaches the given
// conditions, each with a due date computed from now plus its DueInDays.
func (s *Service) ApproveWithConditions(ctx context.Context, id, notes string, inputs []ConditionInput) (register.Entry, []register.Condition, error) {
	if len(inputs) == 0 {
		return register.Entry{}, nil, fmt.Errorf("api: at least one condition is required")
	}
	now := s.now()
	conds := make([]register.Condition, 0, len(inputs))
	for _, in := range inputs {
		if in.Description == "" {
			return register.Entry{}, nil, fmt.Errorf("api: condition description is required")
		}
		conds = append(conds, register.Condition{
			Description: in.Description,
			DueAt:       now.AddDate(0, 0, in.DueInDays),
		})
	}
	entry, err := s.store.UpdateStatus(ctx, id, register.StatusApproved, notes)
	if err != nil {
		return register.Entry{}, nil, err
	}
	saved, err := s.store.AddConditions(ctx, id, conds)
	if err != nil {
		return register.Entry{}, nil, err
	}
	return entry, saved, nil
}

// SatisfyCondition records evidence closing a condition. If every condition on
// the entry is then satisfied, the entry transitions to fully-approved.
func (s *Service) SatisfyCondition(ctx context.Context, entryID, condID, evidence string) (register.Entry, register.Condition, error) {
	cond, err := s.store.SatisfyCondition(ctx, entryID, condID, evidence, s.now())
	if err != nil {
		return register.Entry{}, register.Condition{}, err
	}
	all, err := s.store.Conditions(ctx, entryID)
	if err != nil {
		return register.Entry{}, register.Condition{}, err
	}
	allSatisfied := len(all) > 0
	for _, c := range all {
		if !c.Satisfied {
			allSatisfied = false
			break
		}
	}
	entry, err := s.store.Get(ctx, entryID)
	if err != nil {
		return register.Entry{}, register.Condition{}, err
	}
	if allSatisfied && entry.Status == register.StatusApproved {
		entry, err = s.store.UpdateStatus(ctx, entryID, register.StatusFullyApproved, entry.ReviewerNotes)
		if err != nil {
			return register.Entry{}, register.Condition{}, err
		}
	}
	return entry, cond, nil
}

// Overdue returns entries with unsatisfied conditions past due as of the
// service clock.
func (s *Service) Overdue(ctx context.Context) ([]register.OverdueEntry, error) {
	return s.store.Overdue(ctx, s.now())
}

// Conditions returns all conditions attached to an entry.
func (s *Service) Conditions(ctx context.Context, entryID string) ([]register.Condition, error) {
	return s.store.Conditions(ctx, entryID)
}
