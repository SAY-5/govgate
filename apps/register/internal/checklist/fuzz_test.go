package checklist

import "testing"

// FuzzLoad ensures the checklist YAML loader never panics on arbitrary input
// and that any checklist it accepts also passes validation invariants.
func FuzzLoad(f *testing.F) {
	seeds := []string{
		minimal,
		"",
		"not: [valid",
		"version: 1",
		`version: "1"
name: x
categories: []`,
		`version: "1"
name: x
categories:
  - id: c
    weight: 1
    requirements:
      - id: r
        question: q
        weight: 1
        severity: critical`,
		`version: "1"
name: x
categories:
  - id: c
    weight: -1
    requirements:
      - id: r
        question: q
        weight: 1
        severity: low`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		c, err := Load(data)
		if err != nil {
			return // rejected input is fine
		}
		// Anything accepted must satisfy validation again (idempotent) and have
		// the structural invariants the rest of the system relies on.
		if err := c.Validate(); err != nil {
			t.Fatalf("accepted checklist fails re-validation: %v", err)
		}
		if c.Version == "" || c.Name == "" || len(c.Categories) == 0 {
			t.Fatalf("accepted checklist missing required fields: %+v", c)
		}
		seen := map[string]bool{}
		for _, cat := range c.Categories {
			if cat.Weight <= 0 {
				t.Fatalf("accepted non-positive category weight: %v", cat)
			}
			for _, r := range cat.Requirements {
				if r.Weight <= 0 || !r.Severity.Valid() {
					t.Fatalf("accepted invalid requirement: %+v", r)
				}
				if seen[r.ID] {
					t.Fatalf("accepted duplicate requirement id %q", r.ID)
				}
				seen[r.ID] = true
			}
		}
	})
}
