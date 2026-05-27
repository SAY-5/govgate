package checklist

import (
	"os"
	"path/filepath"
	"testing"
)

const minimal = `
version: "1.0.0"
name: tiny
categories:
  - id: security
    weight: 1
    requirements:
      - id: sec.tls
        question: TLS?
        weight: 1
        severity: high
        auto_check: encryption_in_transit
`

func TestLoadMinimal(t *testing.T) {
	c, err := Load([]byte(minimal))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Name != "tiny" || c.Version != "1.0.0" {
		t.Fatalf("unexpected header: %+v", c)
	}
	if len(c.Categories) != 1 || len(c.Categories[0].Requirements) != 1 {
		t.Fatalf("unexpected shape: %+v", c.Categories)
	}
	r, cat, ok := c.RequirementByID("sec.tls")
	if !ok || cat != "security" || r.AutoCheck != "encryption_in_transit" {
		t.Fatalf("RequirementByID = %+v %q %v", r, cat, ok)
	}
}

func TestAutoChecksSortedUnique(t *testing.T) {
	c, err := Load([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	got := c.AutoChecks()
	if len(got) != 1 || got[0] != "encryption_in_transit" {
		t.Fatalf("AutoChecks = %v", got)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]string{
		"missing version": `name: x
categories: [{id: c, weight: 1, requirements: [{id: r, question: q, weight: 1, severity: low}]}]`,
		"missing name": `version: "1"
categories: [{id: c, weight: 1, requirements: [{id: r, question: q, weight: 1, severity: low}]}]`,
		"no categories": `version: "1"
name: x`,
		"bad severity": `version: "1"
name: x
categories: [{id: c, weight: 1, requirements: [{id: r, question: q, weight: 1, severity: nope}]}]`,
		"zero req weight": `version: "1"
name: x
categories: [{id: c, weight: 1, requirements: [{id: r, question: q, weight: 0, severity: low}]}]`,
		"dup requirement id": `version: "1"
name: x
categories:
  - id: a
    weight: 1
    requirements: [{id: r, question: q, weight: 1, severity: low}]
  - id: b
    weight: 1
    requirements: [{id: r, question: q, weight: 1, severity: low}]`,
		"dup category id": `version: "1"
name: x
categories:
  - id: a
    weight: 1
    requirements: [{id: r1, question: q, weight: 1, severity: low}]
  - id: a
    weight: 1
    requirements: [{id: r2, question: q, weight: 1, severity: low}]`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load([]byte(body)); err == nil {
				t.Fatalf("expected error for %q", name)
			}
		})
	}
}

func TestSeverityRank(t *testing.T) {
	if SeverityCritical.Rank() <= SeverityHigh.Rank() ||
		SeverityHigh.Rank() <= SeverityMedium.Rank() ||
		SeverityMedium.Rank() <= SeverityLow.Rank() {
		t.Fatal("severity ranks not strictly ordered")
	}
	if SeverityValid := Severity("bogus").Valid(); SeverityValid {
		t.Fatal("bogus severity reported valid")
	}
}

func TestLoadShippedChecklists(t *testing.T) {
	dir := repoChecklistDir(t)
	all, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	for _, name := range []string{"default", "strict"} {
		if _, ok := all[name]; !ok {
			t.Fatalf("expected checklist %q in %s", name, dir)
		}
	}
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
	t.Skip("checklists dir not found from test working directory")
	return ""
}
