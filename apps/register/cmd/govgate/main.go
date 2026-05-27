// Command govgate is the entry point for the GovGate register service and CLI.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/SAY-5/govgate/apps/register/internal/api"
	"github.com/SAY-5/govgate/apps/register/internal/checklist"
	"github.com/SAY-5/govgate/apps/register/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	switch os.Args[1] {
	case "serve":
		if err := serve(logger); err != nil {
			logger.Error("serve failed", "err", err)
			os.Exit(1)
		}
	case "benchregress":
		if err := benchRegress(os.Args[2:]); err != nil {
			logger.Error("benchregress failed", "err", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func serve(logger *slog.Logger) error {
	dsn := envOr("GOVGATE_DATABASE_URL", "postgres://govgate:govgate@localhost:5432/govgate?sslmode=disable")
	dir := envOr("GOVGATE_CHECKLIST_DIR", "/checklists")
	addr := envOr("GOVGATE_ADDR", ":8080")

	checklists, err := checklist.LoadDir(dir)
	if err != nil {
		return fmt.Errorf("load checklists: %w", err)
	}
	logger.Info("checklists loaded", "count", len(checklists), "dir", dir)

	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()

	svc, err := api.NewService(st, checklists, "default")
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           svc.Handler(logger),
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Info("listening", "addr", addr)
	return srv.ListenAndServe()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: govgate <serve|benchregress> [flags]")
}
