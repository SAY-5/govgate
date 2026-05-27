package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SAY-5/govgate/apps/register/internal/register"
	"github.com/SAY-5/govgate/apps/register/internal/scoring"
	"github.com/SAY-5/govgate/apps/register/internal/store"
)

// Handler builds the HTTP router for the service.
func (s *Service) Handler(logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	mux := http.NewServeMux()
	h := &handlers{svc: s, log: logger}

	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /v1/submissions", h.submit)
	mux.HandleFunc("GET /v1/register", h.list)
	mux.HandleFunc("GET /v1/register/{id}", h.get)
	mux.HandleFunc("POST /v1/register/{id}/status", h.setStatus)
	return logging(logger, mux)
}

type handlers struct {
	svc *Service
	log *slog.Logger
}

func (h *handlers) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) submit(w http.ResponseWriter, r *http.Request) {
	var in SubmitInput
	if err := decode(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Submission.Name == "" || in.Submission.Vendor == "" {
		writeError(w, http.StatusBadRequest, "submission name and vendor are required")
		return
	}
	entry, err := h.svc.Submit(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	entry, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	q := register.Query{
		Status: register.Status(r.URL.Query().Get("status")),
		Band:   scoring.Band(r.URL.Query().Get("band")),
		Vendor: r.URL.Query().Get("vendor"),
		Cursor: r.URL.Query().Get("cursor"),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			q.Limit = n
		}
	}
	page, err := h.svc.List(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if page.Entries == nil {
		page.Entries = []register.Entry{}
	}
	writeJSON(w, http.StatusOK, page)
}

type statusReq struct {
	Status register.Status `json:"status"`
	Notes  string          `json:"reviewer_notes"`
}

func (h *handlers) setStatus(w http.ResponseWriter, r *http.Request) {
	var req statusReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := h.svc.SetStatus(r.Context(), r.PathValue("id"), req.Status, req.Notes)
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

// --- helpers ---

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func handleStoreErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var nf store.ErrNotFound
	if errors.As(err, &nf) {
		writeError(w, http.StatusNotFound, nf.Error())
		return true
	}
	writeError(w, http.StatusBadRequest, err.Error())
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
