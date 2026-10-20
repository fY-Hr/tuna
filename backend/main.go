package main

import (
	"database/sql"
	"fmt"
	"net/http"
	//gorm
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsnString := "postgres://aeon:tunapassword@localhost:5432/tuna?sslmode=disable"
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
	http.HandleFunc("/api/auth/register", server.registerHandler)
	http.HandleFunc("/api/auth/login", server.loginHandler)
	defer db.Close()

	fmt.Println("Tuna server starting")
	http.HandleFunc("/api/waves", server.waveHandler)
	http.ListenAndServe(":8080", nil)
}
