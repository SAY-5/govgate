package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
)

// baseline records the reference scoring throughput, in assessments/sec.
type baseline struct {
	ScorePerSec float64 `json:"score_per_sec"`
}

// benchRegress measures scoring throughput and compares it against a committed
// baseline, failing if throughput regresses by more than the threshold. With
// --update it (re)writes the baseline instead of gating.
func benchRegress(args []string) error {
	fs := flag.NewFlagSet("benchregress", flag.ContinueOnError)
	threshold := fs.Float64("threshold", 0.30, "max fractional throughput regression before failing")
	count := fs.Int("count", 20000, "number of scoring iterations to time")
	baselinePath := fs.String("baseline", "bench/baseline.json", "baseline JSON path")
	update := fs.Bool("update", false, "write the measured throughput as the new baseline")
	checklistPath := fs.String("checklist", "../../checklists/default.yaml", "checklist YAML path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	c, err := checklist.LoadFile(*checklistPath)
	if err != nil {
		return err
	}
	perSec := measureScoreThroughput(c, *count)
	fmt.Printf("scoring throughput: %.0f assessments/sec (%d iters)\n", perSec, *count)

	if *update {
		return writeBaseline(*baselinePath, baseline{ScorePerSec: perSec})
	}

	base, err := readBaseline(*baselinePath)
	if err != nil {
		return fmt.Errorf("read baseline (run with --update to create): %w", err)
	}
	if base.ScorePerSec <= 0 {
		return fmt.Errorf("invalid baseline throughput %.2f", base.ScorePerSec)
	}
	regression := (base.ScorePerSec - perSec) / base.ScorePerSec
	fmt.Printf("baseline: %.0f/sec, regression: %.1f%% (threshold %.0f%%)\n",
		base.ScorePerSec, regression*100, *threshold*100)
	if regression > *threshold {
		return fmt.Errorf("throughput regressed %.1f%% > %.1f%% threshold",
			regression*100, *threshold*100)
	}
	fmt.Println("benchregress: within threshold")
	return nil
}

func measureScoreThroughput(c *checklist.Checklist, count int) float64 {
	tls := true
	thirty := 30
	pii := false
	sub := scoring.Submission{
		Vendor: "Acme",
		StructuredFields: scoring.StructuredFields{
			DataRegion:          "eu-west-1",
			ApprovedRegions:     []string{"eu-west-1"},
			ProcessesPII:        &pii,
			RetentionDays:       &thirty,
			EncryptionInTransit: &tls,
		},
	}
	// Warm up.
	for i := 0; i < 1000; i++ {
		_ = scoring.Score(c, sub, nil)
	}
	start := time.Now()
	for i := 0; i < count; i++ {
		_ = scoring.Score(c, sub, nil)
	}
	elapsed := time.Since(start)
	return float64(count) / elapsed.Seconds()
}

func readBaseline(path string) (baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return baseline{}, err
	}
	var b baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return baseline{}, err
	}
	return b, nil
}

func writeBaseline(path string, b baseline) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote baseline %s\n", path)
	return nil
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
