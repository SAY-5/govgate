//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
)

// newTestStore spins up a throwaway Postgres container and returns an open
// store, registering cleanup on the test.
func newTestStore(t *testing.T) *Postgres {
	t.Helper()
	return openContainerStore(context.Background(), t)
}

// openContainerStore starts a throwaway Postgres and opens a store against it,
// registering termination/close cleanup. It accepts testing.TB so both tests
// and benchmarks can use it.
func openContainerStore(ctx context.Context, tb testing.TB) *Postgres {
	tb.Helper()
	ctr, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		tcpostgres.WithDatabase("govgate"),
		tcpostgres.WithUsername("govgate"),
		tcpostgres.WithPassword("govgate"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		tb.Fatalf("start postgres: %v", err)
	}
	tb.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		tb.Fatalf("connection string: %v", err)
	}
	st, err := Open(ctx, dsn)
	if err != nil {
		tb.Fatalf("open store: %v", err)
	}
	tb.Cleanup(st.Close)
	return st
}

func sampleEntry(vendor, name string, band scoring.Band, status register.Status) register.Entry {
	return register.Entry{
		Submission:       scoring.Submission{Name: name, Vendor: vendor, Description: "d"},
		ChecklistName:    "default",
		ChecklistVersion: "1.0.0",
		Assessment:       scoring.Assessment{OverallBand: band, OverallScore: 0.8},
		Status:           status,
	}
}

func TestInsertGetRoundTrip(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	in := sampleEntry("Acme", "Chat", scoring.BandMedium, register.StatusPending)
	saved, err := st.Insert(ctx, in)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if saved.ID == "" || saved.CreatedAt.IsZero() {
		t.Fatalf("server fields not set: %+v", saved)
	}
	got, err := st.Get(ctx, saved.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Submission.Vendor != "Acme" || got.Assessment.OverallBand != scoring.BandMedium {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestGetNotFound(t *testing.T) {
	st := newTestStore(t)
	_, err := st.Get(context.Background(), "00000000-0000-0000-0000-000000000000")
	if _, ok := err.(ErrNotFound); !ok {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListFiltersAndPaginates(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	seed := []register.Entry{
		sampleEntry("Acme", "A", scoring.BandLow, register.StatusApproved),
		sampleEntry("Acme", "B", scoring.BandHigh, register.StatusPending),
		sampleEntry("Globex", "C", scoring.BandHigh, register.StatusPending),
		sampleEntry("Globex", "D", scoring.BandCritical, register.StatusRejected),
	}
	for _, e := range seed {
		if _, err := st.Insert(ctx, e); err != nil {
			t.Fatalf("seed insert: %v", err)
		}
		time.Sleep(2 * time.Millisecond) // distinct created_at ordering
	}

	byVendor, err := st.List(ctx, register.Query{Vendor: "Acme"})
	if err != nil || len(byVendor.Entries) != 2 {
		t.Fatalf("vendor filter: %d entries err=%v", len(byVendor.Entries), err)
	}
	byBand, err := st.List(ctx, register.Query{Band: scoring.BandHigh})
	if err != nil || len(byBand.Entries) != 2 {
		t.Fatalf("band filter: %d entries err=%v", len(byBand.Entries), err)
	}
	byStatus, err := st.List(ctx, register.Query{Status: register.StatusPending})
	if err != nil || len(byStatus.Entries) != 2 {
		t.Fatalf("status filter: %d entries err=%v", len(byStatus.Entries), err)
	}

	// Pagination: limit 2 over 4 rows yields a cursor and the rest.
	p1, err := st.List(ctx, register.Query{Limit: 2})
	if err != nil || len(p1.Entries) != 2 || p1.NextCursor == "" {
		t.Fatalf("page1: %d entries cursor=%q err=%v", len(p1.Entries), p1.NextCursor, err)
	}
	p2, err := st.List(ctx, register.Query{Limit: 2, Cursor: p1.NextCursor})
	if err != nil || len(p2.Entries) != 2 {
		t.Fatalf("page2: %d entries err=%v", len(p2.Entries), err)
	}
	if p1.Entries[0].ID == p2.Entries[0].ID {
		t.Fatal("pagination returned overlapping entries")
	}
}

func TestUpdateStatus(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	saved, err := st.Insert(ctx, sampleEntry("Acme", "X", scoring.BandLow, register.StatusPending))
	if err != nil {
		t.Fatal(err)
	}
	upd, err := st.UpdateStatus(ctx, saved.ID, register.StatusApproved, "looks fine")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if upd.Status != register.StatusApproved || upd.ReviewerNotes != "looks fine" {
		t.Fatalf("update mismatch: %+v", upd)
	}
}
