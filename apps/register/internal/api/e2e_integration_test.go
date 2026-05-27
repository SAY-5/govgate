//go:build integration

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/register"
)

// TestEndToEndExamples submits each shipped example through the HTTP API and
// asserts the persisted assessment matches the committed golden assessment.
// This ties the scoring engine, the register store, and the example fixtures
// together over the wire.
func TestEndToEndExamples(t *testing.T) {
	root := repoRoot(t)
	checklists, err := checklist.LoadDir(filepath.Join(root, "checklists"))
	if err != nil {
		t.Fatal(err)
	}
	srv := newServerWith(t, checklists, "default")

	cases := []struct {
		file      string
		wantBand  string
		wantCrits int
	}{
		{"acme_chat", "low", 0},
		{"globex_vision", "critical", 1},
		{"initech_summarizer", "low", 0},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			subPath := filepath.Join(root, "examples", "submissions", tc.file+".json")
			raw, err := os.ReadFile(subPath)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.Post(srv.URL+"/v1/submissions", "application/json", bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("submit %s: status %d", tc.file, resp.StatusCode)
			}
			var entry register.Entry
			if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
				t.Fatal(err)
			}
			if string(entry.Assessment.OverallBand) != tc.wantBand {
				t.Fatalf("%s: band %s want %s", tc.file, entry.Assessment.OverallBand, tc.wantBand)
			}
			if len(entry.Assessment.Criticals) != tc.wantCrits {
				t.Fatalf("%s: criticals %v want %d", tc.file, entry.Assessment.Criticals, tc.wantCrits)
			}

			// Fetch from the register and confirm it round-trips.
			got, err := http.Get(srv.URL + "/v1/register/" + entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			defer got.Body.Close()
			var fetched register.Entry
			if err := json.NewDecoder(got.Body).Decode(&fetched); err != nil {
				t.Fatal(err)
			}
			if fetched.Assessment.OverallBand != entry.Assessment.OverallBand {
				t.Fatalf("%s: fetched band %s != submitted %s", tc.file,
					fetched.Assessment.OverallBand, entry.Assessment.OverallBand)
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "checklists")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("repo root not found")
	return ""
}
