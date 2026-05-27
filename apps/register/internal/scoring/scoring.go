package scoring

import (
	"sort"
	"strings"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
)

// Band is the discrete risk classification for a category or the overall tool.
type Band string

const (
	BandLow      Band = "low"
	BandMedium   Band = "medium"
	BandHigh     Band = "high"
	BandCritical Band = "critical"
)

var bandRank = map[Band]int{BandLow: 0, BandMedium: 1, BandHigh: 2, BandCritical: 3}

// Rank returns the ordinal of the band (low=0 .. critical=3).
func (b Band) Rank() int { return bandRank[b] }

// atLeast returns the higher-risk of two bands.
func atLeast(a, b Band) Band {
	if bandRank[a] >= bandRank[b] {
		return a
	}
	return b
}

// RequirementResult is the evaluated outcome of one requirement.
type RequirementResult struct {
	ID       string             `json:"id"`
	Category string             `json:"category"`
	Question string             `json:"question"`
	Weight   float64            `json:"weight"`
	Severity checklist.Severity `json:"severity"`
	Auto     bool               `json:"auto"`
	Outcome  Outcome            `json:"outcome"`
}

// Failed reports whether the requirement did not pass (fail or unknown).
func (r RequirementResult) Failed() bool { return r.Outcome != OutcomePass }

// CategoryResult is the rolled-up score for one category.
type CategoryResult struct {
	ID      string              `json:"id"`
	Score   float64             `json:"score"` // weighted fraction satisfied, 0..1
	Band    Band                `json:"band"`
	Results []RequirementResult `json:"results"`
}

// Assessment is the full scored result of a submission against a checklist.
type Assessment struct {
	ChecklistName    string           `json:"checklist_name"`
	ChecklistVersion string           `json:"checklist_version"`
	OverallScore     float64          `json:"overall_score"` // weighted, 0..1
	OverallBand      Band             `json:"overall_band"`
	Categories       []CategoryResult `json:"categories"`
	Criticals        []string         `json:"critical_failures"` // requirement ids
}

// Score evaluates a submission against a checklist, applying auto-checks where
// present and human/LLM judgments otherwise. The overall band is the weighted
// aggregate gated by severity: any failed critical requirement caps the overall
// at high or worse.
func Score(c *checklist.Checklist, sub Submission, judgments Judgments) Assessment {
	a := Assessment{
		ChecklistName:    c.Name,
		ChecklistVersion: c.Version,
	}

	var totalCatWeight float64
	var weightedScoreSum float64
	worstCriticalBand := BandLow

	for _, cat := range c.Categories {
		cr := CategoryResult{ID: cat.ID}
		var reqWeight, satisfiedWeight float64

		for _, r := range cat.Requirements {
			res := RequirementResult{
				ID:       r.ID,
				Category: cat.ID,
				Question: r.Question,
				Weight:   r.Weight,
				Severity: r.Severity,
			}
			res.Outcome = evaluate(r, sub, judgments)
			res.Auto = r.AutoCheck != ""

			reqWeight += r.Weight
			if res.Outcome == OutcomePass {
				satisfiedWeight += r.Weight
			} else if r.Severity == checklist.SeverityCritical {
				a.Criticals = append(a.Criticals, r.ID)
				worstCriticalBand = atLeast(worstCriticalBand, BandHigh)
			}
			cr.Results = append(cr.Results, res)
		}

		if reqWeight > 0 {
			cr.Score = satisfiedWeight / reqWeight
		}
		cr.Band = scoreToBand(cr.Score)
		a.Categories = append(a.Categories, cr)

		weightedScoreSum += cat.Weight * cr.Score
		totalCatWeight += cat.Weight
	}

	if totalCatWeight > 0 {
		a.OverallScore = weightedScoreSum / totalCatWeight
	}
	a.OverallBand = atLeast(scoreToBand(a.OverallScore), worstCriticalBand)
	sort.Strings(a.Criticals)
	return a
}

// evaluate resolves a single requirement's outcome.
func evaluate(r checklist.Requirement, sub Submission, judgments Judgments) Outcome {
	switch {
	case r.AutoCheck == "vendor_named":
		return boolToOutcome(strings.TrimSpace(sub.Vendor) != "")
	case r.AutoCheck != "":
		if fn, ok := autoChecks[r.AutoCheck]; ok {
			return fn(sub.StructuredFields)
		}
		// Unknown auto-check key: fail closed rather than silently pass.
		return OutcomeUnknown
	default:
		if o, ok := judgments[r.ID]; ok {
			return o
		}
		return OutcomeUnknown
	}
}

// scoreToBand maps a 0..1 satisfaction score to a risk band. Higher
// satisfaction means lower risk.
func scoreToBand(score float64) Band {
	switch {
	case score >= 0.90:
		return BandLow
	case score >= 0.70:
		return BandMedium
	case score >= 0.40:
		return BandHigh
	default:
		return BandCritical
	}
}
