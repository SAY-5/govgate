//go:build integration

package api

import (
	"context"
	"testing"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
)

// v1Checklist: a lenient policy the sample tool passes cleanly.
const clV1 = `
version: "1.0.0"
name: govern
categories:
  - id: security
    weight: 1
    requirements:
      - id: sec.tls
        question: tls?
        weight: 1
        severity: high
        auto_check: encryption_in_transit
`

// v2Checklist: adds a critical no-customer-training requirement the tool fails.
const clV2 = `
version: "2.0.0"
name: govern
categories:
  - id: security
    weight: 1
    requirements:
      - id: sec.tls
        question: tls?
        weight: 1
        severity: high
        auto_check: encryption_in_transit
  - id: model_provenance
    weight: 2
    requirements:
      - id: mp.no_customer_training
        question: excludes customer data from training?
        weight: 3
        severity: critical
        auto_check: no_customer_training
`

func TestReassessOnPolicyChange(t *testing.T) {
	ctx := context.Background()

	v1, err := checklist.Load([]byte(clV1))
	if err != nil {
		t.Fatal(err)
	}
	st := openStore(t)
	svc, err := NewService(st, map[string]*checklist.Checklist{"govern": v1}, "govern")
	if err != nil {
		t.Fatal(err)
	}

	// Submit a tool that passes under v1: TLS on, does not train on customer data.
	tls := true
	trains := true // it DOES train on customer data, which v2 will penalize
	var in SubmitInput
	in.Submission.Name = "Trainer"
	in.Submission.Vendor = "Acme"
	in.Submission.StructuredFields.EncryptionInTransit = &tls
	in.Submission.StructuredFields.TrainsOnCustomerData = &trains

	entry, err := svc.Submit(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Assessment.OverallBand != scoring.BandLow {
		t.Fatalf("under v1 expected low, got %s", entry.Assessment.OverallBand)
	}
	if entry.Stale {
		t.Fatal("fresh entry should not be stale")
	}

	// Publish v2: adds a critical requirement the tool fails.
	v2, err := checklist.Load([]byte(clV2))
	if err != nil {
		t.Fatal(err)
	}
	flagged, err := svc.PublishChecklist(ctx, v2)
	if err != nil {
		t.Fatal(err)
	}
	if flagged != 1 {
		t.Fatalf("expected 1 entry flagged stale, got %d", flagged)
	}

	// The entry should now be stale.
	got, err := svc.Get(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Stale {
		t.Fatal("entry should be stale after policy change")
	}

	// Reassess under v2: band must rise to critical.
	updated, diff, err := svc.Reassess(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Assessment.OverallBand != scoring.BandCritical {
		t.Fatalf("after reassess expected critical, got %s", updated.Assessment.OverallBand)
	}
	if updated.Stale {
		t.Fatal("reassessed entry should no longer be stale")
	}
	if !diff.BandRose || diff.FromBand != scoring.BandLow || diff.ToBand != scoring.BandCritical {
		t.Fatalf("diff did not capture the rise: %+v", diff)
	}
	if len(diff.NewCriticals) != 1 || diff.NewCriticals[0] != "mp.no_customer_training" {
		t.Fatalf("expected new critical mp.no_customer_training, got %v", diff.NewCriticals)
	}

	// Both assessments must be retained in history, oldest first.
	hist, err := svc.History(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 {
		t.Fatalf("expected 2 history records, got %d", len(hist))
	}
	if hist[0].ChecklistVersion != "1.0.0" || hist[1].ChecklistVersion != "2.0.0" {
		t.Fatalf("history versions wrong: %s then %s", hist[0].ChecklistVersion, hist[1].ChecklistVersion)
	}
	if hist[0].Assessment.OverallBand != scoring.BandLow || hist[1].Assessment.OverallBand != scoring.BandCritical {
		t.Fatalf("history bands wrong: %s then %s",
			hist[0].Assessment.OverallBand, hist[1].Assessment.OverallBand)
	}
}

var _ = register.StatusPending // keep register import used across build tags
