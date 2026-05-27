package scoring

import "testing"

// BenchmarkScore measures scoring time for a single submission against a
// realistic multi-category checklist.
func BenchmarkScore(b *testing.B) {
	c := fuzzChecklist()
	tls := true
	thirty := 30
	pii := false
	sub := Submission{
		Vendor: "Acme",
		StructuredFields: StructuredFields{
			DataRegion:          "eu-west-1",
			ApprovedRegions:     []string{"eu-west-1"},
			ProcessesPII:        &pii,
			RetentionDays:       &thirty,
			EncryptionInTransit: &tls,
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Score(c, sub, nil)
	}
}
