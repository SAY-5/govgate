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
	mux.HandleFunc("GET /v1/register/overdue", h.overdue)
	mux.HandleFunc("GET /v1/register/{id}", h.get)
	mux.HandleFunc("POST /v1/register/{id}/status", h.setStatus)
	mux.HandleFunc("POST /v1/register/{id}/reassess", h.reassess)
	mux.HandleFunc("GET /v1/register/{id}/history", h.history)
	mux.HandleFunc("POST /v1/register/{id}/approve-with-conditions", h.approveWithConditions)
	mux.HandleFunc("GET /v1/register/{id}/conditions", h.listConditions)
	mux.HandleFunc("POST /v1/register/{id}/conditions/{cid}/satisfy", h.satisfyCondition)
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

func (h *handlers) reassess(w http.ResponseWriter, r *http.Request) {
	entry, diff, err := h.svc.Reassess(r.Context(), r.PathValue("id"))
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": entry, "diff": diff})
}

func (h *handlers) history(w http.ResponseWriter, r *http.Request) {
	records, err := h.svc.History(r.Context(), r.PathValue("id"))
	if handleStoreErr(w, err) {
		return
	}
	if records == nil {
		records = []register.AssessmentRecord{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": records})
}

func (h *handlers) overdue(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.Overdue(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []register.OverdueEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"overdue": entries})
}

type approveConditionsReq struct {
	ReviewerNotes string           `json:"reviewer_notes"`
	Conditions    []ConditionInput `json:"conditions"`
}

func (h *handlers) approveWithConditions(w http.ResponseWriter, r *http.Request) {
	var req approveConditionsReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entry, conds, err := h.svc.ApproveWithConditions(r.Context(), r.PathValue("id"), req.ReviewerNotes, req.Conditions)
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": entry, "conditions": conds})
}

func (h *handlers) listConditions(w http.ResponseWriter, r *http.Request) {
	conds, err := h.svc.Conditions(r.Context(), r.PathValue("id"))
	if handleStoreErr(w, err) {
		return
	}
	if conds == nil {
		conds = []register.Condition{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"conditions": conds})
}

type satisfyReq struct {
	Evidence string `json:"evidence"`
}

func (h *handlers) satisfyCondition(w http.ResponseWriter, r *http.Request) {
	var req satisfyReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entry, cond, err := h.svc.SatisfyCondition(r.Context(), r.PathValue("id"), r.PathValue("cid"), req.Evidence)
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entry": entry, "condition": cond})
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
