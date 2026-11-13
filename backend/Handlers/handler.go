package Handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Server struct {
	db        *sql.DB
	jwtSecret []byte
}

func NewServer(db *sql.DB, jwtSecret []byte) *Server {
	return &Server{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}
