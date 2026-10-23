package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/waves", s.getWavesHandler)
	r.Get("/api/waves/{id}", s.getWavesByIDHandler)
	r.Post("/api/auth/register", s.registerHandler)
	r.Post("/api/auth/login", s.loginHandler)

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Post("/api/waves", s.createWavesHandler)
		r.Delete("/api/waves/{id}", s.deleteWaveHandler)
		r.Post("/api/waves/{id}/react", s.reactWaveHandler)
	})

	return r
}
