package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	//gorm
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsnString := os.Getenv("DATABASE_URL")
	if dsnString == "" {
		dsnString = "postgres://postgre:postgre@localhost:5432/tuna?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsnString)
	if err != nil {
		fmt.Println("Invalid request", err)
		return
	}
	err = db.Ping()
	if err != nil {
		fmt.Println("Failed to connect to database", err)
		return
	}
	fmt.Println("Connected successfully")

	server := &Server{db: db}
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/waves", server.getWavesHandler)
	r.Post("/api/waves", server.createWavesHandler)
	r.Post("/api/auth/register", server.registerHandler)
	r.Post("/api/auth/login", server.loginHandler)
	defer db.Close()

	fmt.Println("Tuna server starting")
	http.ListenAndServe(":8080", r)
}
