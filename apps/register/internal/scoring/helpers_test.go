package scoring

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
)

var fuzzClOnce struct {
	sync.Once
	cl *checklist.Checklist
}

// fuzzChecklist returns a fixed checklist exercising every auto-check, built
// once for fuzz iterations.
func fuzzChecklist() *checklist.Checklist {
	fuzzClOnce.Do(func() {
		c, err := checklist.Load([]byte(`
version: "1"
name: fuzz
categories:
  - id: data_residency
    weight: 1
    requirements:
      - id: dr.present
        question: q
        weight: 1
        severity: medium
        auto_check: data_region_present
      - id: dr.approved
        question: q
        weight: 1
        severity: high
        auto_check: data_region_approved
  - id: pii
    weight: 2
    requirements:
      - id: pii.unmasked
        question: q
        weight: 1
        severity: critical
        auto_check: pii_not_unmasked
  - id: retention
    weight: 1
    requirements:
      - id: ret.bounded
        question: q
        weight: 1
        severity: high
        auto_check: retention_bounded
`))
		if err != nil {
			panic(err)
		}
		fuzzClOnce.cl = c
	})
	return fuzzClOnce.cl
}

// repoChecklistDir walks up to find the repo-level checklists directory.
func repoChecklistDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		cand := filepath.Join(dir, "checklists")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("checklists dir not found")
	return ""
}
