package scoring

import (
	"math/rand"
	"testing"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
)

// randChecklist builds a checklist with a deterministic-from-seed mix of
// categories, weights, and severities. Every requirement uses a judgment-based
// (non-auto) resolution so the test can drive outcomes directly.
func randChecklist(rng *rand.Rand) *checklist.Checklist {
	nCat := 1 + rng.Intn(4)
	c := &checklist.Checklist{Version: "1", Name: "rand"}
	id := 0
	for ci := 0; ci < nCat; ci++ {
		cat := checklist.Category{
			ID:     catID(ci),
			Weight: 1 + rng.Float64()*4,
		}
		nReq := 1 + rng.Intn(4)
		for ri := 0; ri < nReq; ri++ {
			id++
			cat.Requirements = append(cat.Requirements, checklist.Requirement{
				ID:       reqID(id),
				Question: "q",
				Weight:   1 + rng.Float64()*4,
				Severity: randSeverity(rng),
			})
		}
		c.Categories = append(c.Categories, cat)
	}
	return c
}

func catID(i int) string { return "cat" + itoa(i) }
func reqID(i int) string { return "req" + itoa(i) }
func itoa(i int) string  { return string(rune('a'+i%26)) + string(rune('0'+(i/26)%10)) }

func randSeverity(rng *rand.Rand) checklist.Severity {
	switch rng.Intn(4) {
	case 0:
		return checklist.SeverityLow
	case 1:
		return checklist.SeverityMedium
	case 2:
		return checklist.SeverityHigh
	default:
		return checklist.SeverityCritical
	}
}

// allReqIDs returns every requirement id in the checklist.
func allReqIDs(c *checklist.Checklist) []string {
	var ids []string
	for _, cat := range c.Categories {
		for _, r := range cat.Requirements {
			ids = append(ids, r.ID)
		}
	}
	return ids
}

// judgmentsFor builds judgments marking the given ids as pass and the rest fail.
func judgmentsFor(c *checklist.Checklist, passing map[string]bool) Judgments {
	j := Judgments{}
	for _, id := range allReqIDs(c) {
		if passing[id] {
			j[id] = OutcomePass
		} else {
			j[id] = OutcomeFail
		}
	}
	return j
}

// TestPropertyScoresInUnitInterval: category and overall scores are always in
// [0,1] for any random checklist and any random outcome assignment.
func TestPropertyScoresInUnitInterval(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for iter := 0; iter < 500; iter++ {
		c := randChecklist(rng)
		passing := map[string]bool{}
		for _, id := range allReqIDs(c) {
			if rng.Intn(2) == 0 {
				passing[id] = true
			}
		}
		a := Score(c, Submission{Vendor: "v"}, judgmentsFor(c, passing))
		if a.OverallScore < 0 || a.OverallScore > 1.0000001 {
			t.Fatalf("iter %d overall %f out of [0,1]", iter, a.OverallScore)
		}
		for _, cat := range a.Categories {
			if cat.Score < 0 || cat.Score > 1.0000001 {
				t.Fatalf("iter %d cat %s score %f out of [0,1]", iter, cat.ID, cat.Score)
			}
		}
	}
}

// TestPropertyOverallMonotonicInFailures: flipping any single requirement from
// pass to fail never lowers the overall risk band (risk is monotonic in
// failures).
func TestPropertyOverallMonotonicInFailures(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for iter := 0; iter < 500; iter++ {
		c := randChecklist(rng)
		ids := allReqIDs(c)
		passing := map[string]bool{}
		for _, id := range ids {
			passing[id] = true
		}
		base := Score(c, Submission{Vendor: "v"}, judgmentsFor(c, passing))

		// Flip one passing requirement to fail; band must not improve.
		victim := ids[rng.Intn(len(ids))]
		passing[victim] = false
		worse := Score(c, Submission{Vendor: "v"}, judgmentsFor(c, passing))

		if worse.OverallBand.Rank() < base.OverallBand.Rank() {
			t.Fatalf("iter %d: failing %q lowered band %s -> %s",
				iter, victim, base.OverallBand, worse.OverallBand)
		}
		if worse.OverallScore > base.OverallScore+1e-9 {
			t.Fatalf("iter %d: failing %q raised score %f -> %f",
				iter, victim, base.OverallScore, worse.OverallScore)
		}
	}
}

// TestPropertyCriticalFailureCapsHigh: whenever at least one critical
// requirement fails, the overall band is >= high, regardless of other outcomes.
func TestPropertyCriticalFailureCapsHigh(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for iter := 0; iter < 500; iter++ {
		c := randChecklist(rng)
		ids := allReqIDs(c)
		passing := map[string]bool{}
		for _, id := range ids {
			passing[id] = rng.Intn(2) == 0
		}
		a := Score(c, Submission{Vendor: "v"}, judgmentsFor(c, passing))

		anyCriticalFailed := false
		for _, cat := range c.Categories {
			for _, r := range cat.Requirements {
				if r.Severity == checklist.SeverityCritical && !passing[r.ID] {
					anyCriticalFailed = true
				}
			}
		}
		if anyCriticalFailed && a.OverallBand.Rank() < BandHigh.Rank() {
			t.Fatalf("iter %d: critical failed but band is %s", iter, a.OverallBand)
		}
		if !anyCriticalFailed && len(a.Criticals) != 0 {
			t.Fatalf("iter %d: no critical failed but criticals=%v", iter, a.Criticals)
		}
	}
}

// TestPropertyAllPassIsLow: a checklist where every requirement passes always
// yields a low band with no criticals.
func TestPropertyAllPassIsLow(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for iter := 0; iter < 200; iter++ {
		c := randChecklist(rng)
		passing := map[string]bool{}
		for _, id := range allReqIDs(c) {
			passing[id] = true
		}
		a := Score(c, Submission{Vendor: "v"}, judgmentsFor(c, passing))
		if a.OverallBand != BandLow {
			t.Fatalf("iter %d: all-pass band is %s, want low", iter, a.OverallBand)
		}
		if len(a.Criticals) != 0 {
			t.Fatalf("iter %d: all-pass has criticals %v", iter, a.Criticals)
		}
	}
}
