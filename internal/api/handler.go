package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/scott-walker/mememory/internal/bootstrap"
	"github.com/scott-walker/mememory/internal/engine"
	"github.com/scott-walker/mememory/internal/pinned"
)

type Handler struct {
	svc *engine.Service
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.Stats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	memories, err := h.svc.List(r.Context(), engine.ListInput{
		Scope:    q.Get("scope"),
		Project:  q.Get("project"),
		Type:     q.Get("type"),
		Delivery: q.Get("delivery"),
		Limit:    limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	memories, err := h.svc.List(r.Context(), engine.ListInput{Limit: 100})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, m := range memories {
		if m.ID == id {
			writeJSON(w, http.StatusOK, m)
			return
		}
	}
	writeError(w, http.StatusNotFound, "memory not found")
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input engine.RememberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := h.svc.Remember(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	mem, err := h.svc.Update(r.Context(), id, body.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mem)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Forget(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var input engine.RecallInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	results, err := h.svc.Recall(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// List matching memories, then delete each
	memories, err := h.svc.List(r.Context(), engine.ListInput{
		Scope:    q.Get("scope"),
		Project:  q.Get("project"),
		Type:     q.Get("type"),
		Delivery: q.Get("delivery"),
		Limit:    1000,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	deleted := 0
	for _, m := range memories {
		if err := h.svc.Forget(r.Context(), m.ID); err == nil {
			deleted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"deleted": deleted})
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	memories, err := h.svc.List(r.Context(), engine.ListInput{Limit: 10000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=memories.json")
	writeJSON(w, http.StatusOK, memories)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var memories []engine.RememberInput
	if err := json.NewDecoder(r.Body).Decode(&memories); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	imported := 0
	for _, input := range memories {
		if result, err := h.svc.Remember(r.Context(), input); err == nil && result.Memory != nil {
			imported++
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
}

// PinnedPreview renders the full pinned-payload that the UserPromptSubmit
// hook would inject for a given project — used by the admin UI to show
// users exactly what their agent receives every turn.
//
// Rendering uses a fixed seed so the preview is reproducible. The system
// layer rotates with random seeds in production, but for an inspection UI
// stability beats novelty.
func (h *Handler) PinnedPreview(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")

	global, err := h.svc.List(r.Context(), engine.ListInput{
		Scope:    "global",
		Delivery: "pinned",
		Limit:    100,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var projectMems []engine.Memory
	if project != "" {
		projectMems, err = h.svc.List(r.Context(), engine.ListInput{
			Scope:    "project",
			Project:  project,
			Delivery: "pinned",
			Limit:    100,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	markdown := pinned.Format(pinned.Context{
		Project: bootstrap.ProjectInfo{
			Name:   project,
			Source: "admin preview",
		},
		GlobalMems:  global,
		ProjectMems: projectMems,
		Seed:        1,
	})

	resp := struct {
		Markdown string `json:"markdown"`
		Stats    struct {
			Global  int `json:"global"`
			Project int `json:"project"`
			Tokens  int `json:"tokens"`
		} `json:"stats"`
	}{
		Markdown: markdown,
	}
	resp.Stats.Global = len(global)
	resp.Stats.Project = len(projectMems)
	resp.Stats.Tokens = pinned.EstimateTokens(len(markdown))

	writeJSON(w, http.StatusOK, resp)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
