package Handlers

import "database/sql"

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
