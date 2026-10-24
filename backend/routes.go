package main

import (
	"net/http"

	"github.com/fY-Hr/tuna/backend/Handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func setupRoutes(s *Handlers.Server) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/waves", s.GetWavesHandler)
	r.Get("/api/waves/{id}", s.GetWavesByIDHandler)
	r.Post("/api/auth/register", s.RegisterHandler)
	r.Post("/api/auth/login", s.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(s.AuthMiddleware)
		r.Post("/api/waves", s.CreateWavesHandler)
		r.Delete("/api/waves/{id}", s.DeleteWaveHandler)
		r.Post("/api/waves/{id}/react", s.ReactWaveHandler)
	})

	return r
}
