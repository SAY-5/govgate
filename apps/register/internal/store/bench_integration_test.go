//go:build integration

package store

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
)

// BenchmarkInsert measures register write throughput (assessments persisted per
// second) against a real Postgres.
func BenchmarkInsert(b *testing.B) {
	st := newBenchStore(b)
	ctx := context.Background()
	e := register.Entry{
		Submission:       scoring.Submission{Name: "T", Vendor: "Acme"},
		ChecklistName:    "default",
		ChecklistVersion: "1.0.0",
		Assessment:       scoring.Assessment{OverallBand: scoring.BandLow, OverallScore: 0.95},
		Status:           register.StatusPending,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := st.Insert(ctx, e); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListAtScale seeds a large table and reports query latency
// percentiles for a filtered, paginated list.
func BenchmarkListAtScale(b *testing.B) {
	st := newBenchStore(b)
	ctx := context.Background()
	const rows = 5000 // CI-friendly; the gate command can drive higher locally
	seed(b, st, rows)

	b.ResetTimer()
	var samples []time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, err := st.List(ctx, register.Query{Band: scoring.BandHigh, Limit: 50}); err != nil {
			b.Fatal(err)
		}
		samples = append(samples, time.Since(start))
	}
	b.StopTimer()
	reportPercentiles(b, samples)
}

func newBenchStore(tb testing.TB) *Postgres {
	tb.Helper()
	ctx := context.Background()
	st := openContainerStore(ctx, tb)
	return st
}

func seed(tb testing.TB, st *Postgres, n int) {
	tb.Helper()
	ctx := context.Background()
	bands := []scoring.Band{scoring.BandLow, scoring.BandMedium, scoring.BandHigh, scoring.BandCritical}
	for i := 0; i < n; i++ {
		e := register.Entry{
			Submission:       scoring.Submission{Name: fmt.Sprintf("tool-%d", i), Vendor: "Acme"},
			ChecklistName:    "default",
			ChecklistVersion: "1.0.0",
			Assessment:       scoring.Assessment{OverallBand: bands[i%len(bands)], OverallScore: 0.5},
			Status:           register.StatusPending,
		}
		if _, err := st.Insert(ctx, e); err != nil {
			tb.Fatal(err)
		}
	}
}

func reportPercentiles(b *testing.B, samples []time.Duration) {
	if len(samples) == 0 {
		return
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	pct := func(p float64) time.Duration {
		idx := int(p * float64(len(samples)-1))
		return samples[idx]
	}
	b.ReportMetric(float64(pct(0.50).Microseconds()), "p50-us")
	b.ReportMetric(float64(pct(0.95).Microseconds()), "p95-us")
	b.ReportMetric(float64(pct(0.99).Microseconds()), "p99-us")
}
