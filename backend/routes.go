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
	r.Get("/api/waves/{id}/reactions", s.GetWaveReactionsHandler)
	r.Get("/api/users/{username}", s.GetProfileHandler)
	r.Post("/api/auth/register", s.RegisterHandler)
	r.Post("/api/auth/login", s.LoginHandler)
	r.Post("/api/auth/refresh", s.RefreshHandler)

	r.Group(func(r chi.Router) {
		r.Use(s.AuthMiddleware)
		r.Post("/api/waves", s.CreateWavesHandler)
		r.Delete("/api/waves/{id}", s.DeleteWaveHandler)
		r.Post("/api/waves/{id}/react", s.ReactWaveHandler)
		r.Get("/api/waves/{id}/reactions/me", s.GetMyReactionHandler)
		// chi prefers static segments over params, so /api/users/me
		// matches here even though /api/users/{username} is public.
		r.Get("/api/users/me", s.GetMeHandler)
		r.Patch("/api/users/me", s.UpdateMeHandler)
	})

	return r
}
