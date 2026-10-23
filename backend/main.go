package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	//gorm
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	_ = godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsnString := os.Getenv("DATABASE_URL")
	if dsnString == "" {
		log.Fatal("ERROR, Please provide DATABASE_URL environment variable")
		return
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("ERROR, Please provide JWT_SECRET environment variable")
		return
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

	server := &Server{db: db, jwtSecret: []byte(jwtSecret)}
	defer db.Close()

	fmt.Println("Tuna server starting")
	if err := http.ListenAndServe(":"+port, server.routes()); err != nil {
		log.Fatal(err)
	}
}
