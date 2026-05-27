package scoring

import (
	"encoding/json"
	"testing"
)

// FuzzSubmissionParse ensures the submission JSON parser never panics on
// arbitrary input and that any successfully parsed submission scores cleanly
// against a fixed checklist.
func FuzzSubmissionParse(f *testing.F) {
	seeds := []string{
		`{}`,
		`{"name":"x","vendor":"y"}`,
		`{"structured_fields":{"retention_days":30,"processes_pii":true}}`,
		`{"structured_fields":{"approved_regions":["eu-west-1"],"data_region":"eu-west-1"}}`,
		`{"structured_fields":{"retention_days":-5}}`,
		`not json`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var sub Submission
		if err := json.Unmarshal(data, &sub); err != nil {
			return // invalid JSON is fine; we only care about no panics
		}
		// A parsed submission must score without panicking and stay in range.
		a := Score(fuzzChecklist(), sub, nil)
		if a.OverallScore < 0 || a.OverallScore > 1.0000001 {
			t.Fatalf("score out of range: %f", a.OverallScore)
		}
	})
}
