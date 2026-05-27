// Package checklist defines the typed model for GovGate intake checklists and
// loads them from YAML. A checklist is a versioned set of weighted, severity-
// tagged requirements grouped into categories. Some requirements carry an
// auto_check key that the scoring engine evaluates against a submission's
// structured fields; the rest require a human or LLM judgment.
package checklist

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Severity ranks the consequence of a failed requirement. The ordering matters:
// the scoring engine uses it to gate the overall risk band.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

var severityRank = map[Severity]int{
	SeverityLow:      0,
	SeverityMedium:   1,
	SeverityHigh:     2,
	SeverityCritical: 3,
}

// Valid reports whether s is a recognized severity.
func (s Severity) Valid() bool {
	_, ok := severityRank[s]
	return ok
}

// Rank returns the ordinal of the severity (low=0 .. critical=3).
func (s Severity) Rank() int { return severityRank[s] }

// Requirement is a single checkable item within a category.
type Requirement struct {
	ID        string   `yaml:"id"`
	Question  string   `yaml:"question"`
	Weight    float64  `yaml:"weight"`
	Severity  Severity `yaml:"severity"`
	AutoCheck string   `yaml:"auto_check,omitempty"`
}

// Category groups related requirements and carries its own relative weight.
type Category struct {
	ID           string        `yaml:"id"`
	Weight       float64       `yaml:"weight"`
	Requirements []Requirement `yaml:"requirements"`
}

// Checklist is the top-level versioned policy document.
type Checklist struct {
	Version     string     `yaml:"version"`
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Categories  []Category `yaml:"categories"`
}

// Load parses and validates a checklist from raw YAML bytes.
func Load(data []byte) (*Checklist, error) {
	var c Checklist
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("checklist: parse yaml: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadFile reads and parses a checklist from a path on disk.
func LoadFile(path string) (*Checklist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("checklist: read %s: %w", path, err)
	}
	c, err := Load(data)
	if err != nil {
		return nil, fmt.Errorf("checklist %s: %w", filepath.Base(path), err)
	}
	return c, nil
}

// LoadDir loads every *.yaml/*.yml checklist in a directory, keyed by name.
func LoadDir(dir string) (map[string]*Checklist, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("checklist: read dir %s: %w", dir, err)
	}
	out := make(map[string]*Checklist)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		c, err := LoadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if _, dup := out[c.Name]; dup {
			return nil, fmt.Errorf("checklist: duplicate name %q in %s", c.Name, dir)
		}
		out[c.Name] = c
	}
	return out, nil
}

// Validate checks structural invariants: non-empty identifiers, positive
// weights, recognized severities, and unique requirement IDs.
func (c *Checklist) Validate() error {
	if strings.TrimSpace(c.Version) == "" {
		return errors.New("checklist: version is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("checklist: name is required")
	}
	if len(c.Categories) == 0 {
		return errors.New("checklist: at least one category is required")
	}
	seenReq := map[string]bool{}
	seenCat := map[string]bool{}
	for _, cat := range c.Categories {
		if strings.TrimSpace(cat.ID) == "" {
			return errors.New("checklist: category id is required")
		}
		if seenCat[cat.ID] {
			return fmt.Errorf("checklist: duplicate category id %q", cat.ID)
		}
		seenCat[cat.ID] = true
		if cat.Weight <= 0 {
			return fmt.Errorf("checklist: category %q weight must be > 0", cat.ID)
		}
		if len(cat.Requirements) == 0 {
			return fmt.Errorf("checklist: category %q has no requirements", cat.ID)
		}
		for _, r := range cat.Requirements {
			if strings.TrimSpace(r.ID) == "" {
				return fmt.Errorf("checklist: requirement id is required in category %q", cat.ID)
			}
			if seenReq[r.ID] {
				return fmt.Errorf("checklist: duplicate requirement id %q", r.ID)
			}
			seenReq[r.ID] = true
			if r.Weight <= 0 {
				return fmt.Errorf("checklist: requirement %q weight must be > 0", r.ID)
			}
			if !r.Severity.Valid() {
				return fmt.Errorf("checklist: requirement %q has invalid severity %q", r.ID, r.Severity)
			}
		}
	}
	return nil
}

// AutoChecks returns the set of distinct auto_check keys referenced by the
// checklist, sorted for stable output.
func (c *Checklist) AutoChecks() []string {
	set := map[string]bool{}
	for _, cat := range c.Categories {
		for _, r := range cat.Requirements {
			if r.AutoCheck != "" {
				set[r.AutoCheck] = true
			}
		}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// RequirementByID returns the requirement and its owning category id.
func (c *Checklist) RequirementByID(id string) (Requirement, string, bool) {
	for _, cat := range c.Categories {
		for _, r := range cat.Requirements {
			if r.ID == id {
				return r, cat.ID, true
			}
		}
	}
	return Requirement{}, "", false
}
