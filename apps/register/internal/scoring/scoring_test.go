package scoring

import (
	"testing"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
)

func ptrBool(b bool) *bool { return &b }
func ptrInt(i int) *int    { return &i }

// twoCat builds a small checklist: one critical requirement in "a", one
// medium in "b".
func twoCat(t *testing.T) *checklist.Checklist {
	t.Helper()
	c, err := checklist.Load([]byte(`
version: "1.0.0"
name: t
categories:
  - id: a
    weight: 1
    requirements:
      - id: a.crit
        question: critical?
        weight: 1
        severity: critical
        auto_check: encryption_in_transit
  - id: b
    weight: 1
    requirements:
      - id: b.med
        question: medium?
        weight: 1
        severity: medium
        auto_check: model_family_known
`))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAllPassLowRisk(t *testing.T) {
	c := twoCat(t)
	sub := Submission{
		Vendor: "Acme",
		StructuredFields: StructuredFields{
			EncryptionInTransit: ptrBool(true),
			ModelFamily:         "gpt-style",
		},
	}
	a := Score(c, sub, nil)
	if a.OverallBand != BandLow {
		t.Fatalf("expected low, got %s (score %.2f)", a.OverallBand, a.OverallScore)
	}
	if len(a.Criticals) != 0 {
		t.Fatalf("unexpected criticals: %v", a.Criticals)
	}
}

func TestCriticalGateCapsAtHigh(t *testing.T) {
	c := twoCat(t)
	// Fail the critical encryption check; pass everything else.
	sub := Submission{
		Vendor: "Acme",
		StructuredFields: StructuredFields{
			EncryptionInTransit: ptrBool(false),
			ModelFamily:         "gpt-style",
		},
	}
	a := Score(c, sub, nil)
	if a.OverallBand.Rank() < BandHigh.Rank() {
		t.Fatalf("critical failure must cap at >= high, got %s", a.OverallBand)
	}
	if len(a.Criticals) != 1 || a.Criticals[0] != "a.crit" {
		t.Fatalf("expected critical a.crit, got %v", a.Criticals)
	}
}

// TestCriticalGateDecisionTable exercises the gate across the cross product of
// critical-pass and noncritical-pass states.
func TestCriticalGateDecisionTable(t *testing.T) {
	c := twoCat(t)
	cases := []struct {
		name        string
		critPass    bool
		medPass     bool
		minBand     Band
		wantCritLen int
	}{
		{"both pass", true, true, BandLow, 0},
		{"crit pass, med fail", true, false, BandMedium, 0},
		{"crit fail, med pass", false, true, BandHigh, 1},
		{"both fail", false, false, BandHigh, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := StructuredFields{EncryptionInTransit: ptrBool(tc.critPass)}
			if tc.medPass {
				f.ModelFamily = "known"
			}
			a := Score(c, Submission{Vendor: "v", StructuredFields: f}, nil)
			if a.OverallBand.Rank() < tc.minBand.Rank() {
				t.Fatalf("%s: band %s below min %s", tc.name, a.OverallBand, tc.minBand)
			}
			if len(a.Criticals) != tc.wantCritLen {
				t.Fatalf("%s: criticals %v want len %d", tc.name, a.Criticals, tc.wantCritLen)
			}
		})
	}
}

func TestUnknownScoresAsFailure(t *testing.T) {
	c := twoCat(t)
	// No structured fields: encryption unknown (critical), model family empty.
	a := Score(c, Submission{Vendor: "v"}, nil)
	if a.OverallBand.Rank() < BandHigh.Rank() {
		t.Fatalf("unknown critical should cap at >= high, got %s", a.OverallBand)
	}
}

func TestJudgmentsForNonAutoRequirements(t *testing.T) {
	c, err := checklist.Load([]byte(`
version: "1"
name: j
categories:
  - id: oversight
    weight: 1
    requirements:
      - id: ho.human
        question: human in loop?
        weight: 1
        severity: high
`))
	if err != nil {
		t.Fatal(err)
	}
	pass := Score(c, Submission{Vendor: "v"}, Judgments{"ho.human": OutcomePass})
	if pass.OverallBand != BandLow {
		t.Fatalf("judged pass should be low, got %s", pass.OverallBand)
	}
	fail := Score(c, Submission{Vendor: "v"}, Judgments{"ho.human": OutcomeFail})
	if fail.OverallBand != BandCritical {
		t.Fatalf("single failed high requirement, all-weight: expected critical band, got %s", fail.OverallBand)
	}
}

func TestScoreNeverExceedsOne(t *testing.T) {
	all, err := checklist.LoadDir(repoChecklistDir(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := Submission{
		Vendor: "Acme",
		StructuredFields: StructuredFields{
			DataRegion:           "eu-west-1",
			ApprovedRegions:      []string{"eu-west-1"},
			ProcessesPII:         ptrBool(false),
			ModelFamily:          "llama-style",
			RetentionDays:        ptrInt(30),
			EncryptionInTransit:  ptrBool(true),
			Subprocessors:        []string{"cloud-x"},
			TrainsOnCustomerData: ptrBool(false),
		},
	}
	for name, c := range all {
		a := Score(c, sub, fullPass(c))
		if a.OverallScore < 0 || a.OverallScore > 1.0 {
			t.Fatalf("%s overall score out of range: %f", name, a.OverallScore)
		}
		for _, cat := range a.Categories {
			if cat.Score < 0 || cat.Score > 1.0 {
				t.Fatalf("%s/%s category score out of range: %f", name, cat.ID, cat.Score)
			}
		}
	}
}

// fullPass returns judgments marking every non-auto requirement as pass.
func fullPass(c *checklist.Checklist) Judgments {
	j := Judgments{}
	for _, cat := range c.Categories {
		for _, r := range cat.Requirements {
			if r.AutoCheck == "" {
				j[r.ID] = OutcomePass
			}
		}
	}
	return j
}
