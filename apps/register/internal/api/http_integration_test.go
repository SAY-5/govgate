//go:build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/store"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	cl, err := checklist.Load([]byte(`
version: "1.0.0"
name: default
categories:
  - id: security
    weight: 1
    requirements:
      - id: sec.tls
        question: tls?
        weight: 1
        severity: critical
        auto_check: encryption_in_transit
`))
	if err != nil {
		t.Fatal(err)
	}
	return newServerWith(t, map[string]*checklist.Checklist{"default": cl}, "default")
}

// newServerWith starts a throwaway Postgres and an httptest server backed by it,
// using the provided checklists.
func newServerWith(t *testing.T, checklists map[string]*checklist.Checklist, def string) *httptest.Server {
	t.Helper()
	st := openStore(t)
	svc, err := NewService(st, checklists, def)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(svc.Handler(nil))
	t.Cleanup(srv.Close)
	return srv
}

// openStore starts a throwaway Postgres and opens a store against it.
func openStore(t *testing.T) store.Store {
	t.Helper()
	ctx := context.Background()
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
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return st
}

func TestSubmitFetchFlow(t *testing.T) {
	srv := newServer(t)
	tls := true
	var body SubmitInput
	body.Submission.Name = "Acme Chat"
	body.Submission.Vendor = "Acme"
	body.Submission.StructuredFields.EncryptionInTransit = &tls

	buf, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/v1/submissions", "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("submit status: %d", resp.StatusCode)
	}
	var entry register.Entry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if entry.ID == "" || entry.Assessment.OverallBand == "" {
		t.Fatalf("bad entry: %+v", entry)
	}

	// Fetch it back.
	got, err := http.Get(srv.URL + "/v1/register/" + entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != http.StatusOK {
		t.Fatalf("get status: %d", got.StatusCode)
	}
	got.Body.Close()

	// List should contain it.
	list, err := http.Get(srv.URL + "/v1/register?vendor=Acme")
	if err != nil {
		t.Fatal(err)
	}
	var page register.Page
	json.NewDecoder(list.Body).Decode(&page)
	list.Body.Close()
	if len(page.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(page.Entries))
	}
}

func TestGetMissingReturns404(t *testing.T) {
	srv := newServer(t)
	resp, err := http.Get(srv.URL + "/v1/register/00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
