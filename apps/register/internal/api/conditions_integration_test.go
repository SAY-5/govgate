//go:build integration

package api

import (
	"context"
	"testing"
	"time"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/register"
)

func newConditionService(t *testing.T) (*Service, func(time.Time)) {
	t.Helper()
	cl, err := checklist.Load([]byte(clV1))
	if err != nil {
		t.Fatal(err)
	}
	st := openStore(t)
	svc, err := NewService(st, map[string]*checklist.Checklist{"govern": cl}, "govern")
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	set := func(at time.Time) { svc.SetClock(func() time.Time { return at }) }
	set(clock)
	return svc, set
}

func submitOne(t *testing.T, svc *Service, name string) register.Entry {
	t.Helper()
	tls := true
	var in SubmitInput
	in.Submission.Name = name
	in.Submission.Vendor = "Acme"
	in.Submission.StructuredFields.EncryptionInTransit = &tls
	entry, err := svc.Submit(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func TestConditionalApprovalLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, setClock := newConditionService(t)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	entry := submitOne(t, svc, "Acme Chat")

	// Approve with a single condition due in 90 days.
	_, conds, err := svc.ApproveWithConditions(ctx, entry.ID, "ok pending PII masking", []ConditionInput{
		{Description: "Mask PII before send; re-review in 90 days.", DueInDays: 90},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(conds) != 1 || conds[0].Satisfied {
		t.Fatalf("expected 1 unsatisfied condition, got %+v", conds)
	}
	condID := conds[0].ID

	// Not yet overdue at start.
	if od, err := svc.Overdue(ctx); err != nil || len(od) != 0 {
		t.Fatalf("expected none overdue at start, got %d err=%v", len(od), err)
	}

	// Advance the clock past the due date.
	setClock(start.AddDate(0, 0, 91))
	od, err := svc.Overdue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(od) != 1 || od[0].Entry.ID != entry.ID {
		t.Fatalf("expected entry overdue, got %+v", od)
	}
	if len(od[0].Conditions) != 1 || od[0].Conditions[0].ID != condID {
		t.Fatalf("overdue condition mismatch: %+v", od[0].Conditions)
	}

	// Satisfy the condition; entry should transition to fully-approved and clear.
	updated, cond, err := svc.SatisfyCondition(ctx, entry.ID, condID, "PII masking deployed; ticket OPS-42")
	if err != nil {
		t.Fatal(err)
	}
	if !cond.Satisfied || cond.SatisfiedAt == nil || cond.Evidence == "" {
		t.Fatalf("condition not properly satisfied: %+v", cond)
	}
	if updated.Status != register.StatusFullyApproved {
		t.Fatalf("expected fully-approved, got %s", updated.Status)
	}

	// No longer overdue.
	if od, err := svc.Overdue(ctx); err != nil || len(od) != 0 {
		t.Fatalf("expected none overdue after satisfy, got %d err=%v", len(od), err)
	}
}

func TestPartialSatisfactionDoesNotTransition(t *testing.T) {
	ctx := context.Background()
	svc, _ := newConditionService(t)
	entry := submitOne(t, svc, "Two Conditions")

	_, conds, err := svc.ApproveWithConditions(ctx, entry.ID, "", []ConditionInput{
		{Description: "Condition A", DueInDays: 30},
		{Description: "Condition B", DueInDays: 60},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Satisfy only the first; entry must remain approved, not fully-approved.
	updated, _, err := svc.SatisfyCondition(ctx, entry.ID, conds[0].ID, "done A")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != register.StatusApproved {
		t.Fatalf("expected still approved with one open condition, got %s", updated.Status)
	}

	// Satisfy the second; now it transitions.
	final, _, err := svc.SatisfyCondition(ctx, entry.ID, conds[1].ID, "done B")
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != register.StatusFullyApproved {
		t.Fatalf("expected fully-approved after all satisfied, got %s", final.Status)
	}
}
