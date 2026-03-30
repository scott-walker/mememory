package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/scott/claude-memory/internal/memory"
)

func NewRouter(svc *memory.Service) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	h := &Handler{svc: svc}

	r.Route("/api", func(r chi.Router) {
		r.Get("/stats", h.Stats)

		r.Route("/memories", func(r chi.Router) {
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Delete("/", h.BulkDelete)
			r.Post("/search", h.Search)
			r.Post("/export", h.Export)
			r.Post("/import", h.Import)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.Get)
				r.Put("/", h.Update)
				r.Delete("/", h.Delete)
			})
		})
	})

	return r
}
