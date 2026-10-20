package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	//gorm
	_ "github.com/jackc/pgx/v5/stdlib"
)

var jwtSecret = []byte("super_secret_tuna_key_replace_later")

func (s *Server) waveHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		// myWave := Wave{
		// 	ID:        1,
		// 	Content:   "Caught my first tuna",
		// 	CreatedAt: time.Now(),
		// }
		query := ` select id, content, created_at, is_archive
	 				from waves order by id desc;`
		rows, err := s.db.Query(query)

		if err != nil {
			http.Error(w, "Failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		waves := []Wave{}

		for rows.Next() {
			var a Wave
			err := rows.Scan(&a.ID, &a.Content, &a.CreatedAt, &a.IsArchive)
			if err != nil {
				http.Error(w, "Failed", http.StatusInternalServerError)
				return
			}
			waves = append(waves, a)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(waves)

	case http.MethodPost:
		var newWave Wave
		err := json.NewDecoder(r.Body).Decode(&newWave)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			return jwtSecret, nil
		})
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID := int64(claims["user_id"].(float64))

		query := ` insert into waves (user_id, content)
			values ($1, $2)
			returning id, created_at, is_archive; `

		err = s.db.QueryRow(query, userID, newWave.Content).Scan(&newWave.ID, &newWave.CreatedAt, &newWave.IsArchive)

		if err != nil {
			fmt.Println("Database Error: ", err)
			http.Error(w, "Failed to save wave", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newWave)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid body request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" || req.Email == "" {
		http.Error(w, "Username, email, and password are required", http.StatusBadRequest)
		return
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	var newID int64
	var createdAt time.Time

	query := `insert into users (username, email, password_hash) 
    values ($1, $2, $3)
    returning id, created_at;`

	err = s.db.QueryRow(query, req.Username, req.Email, string(hashedBytes)).Scan(&newID, &createdAt)

	if err != nil {
		fmt.Println("Database error: ", err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"id":         newID,
		"username":   req.Username,
		"email":      req.Email,
		"created_at": createdAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid body request", http.StatusBadRequest)
		return
	}
	var newID int64
	var storedHash []byte
	query := `select id, password_hash from users where username = $1`
	err = s.db.QueryRow(query, req.Username).Scan(&newID, &storedHash)
	if err != nil {
		http.Error(w, "Failed to find user", http.StatusUnauthorized)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	claims := jwt.MapClaims{
		"user_id": newID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)

	if err != nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	//to accept user
	response := map[string]any{
		"id":       newID,
		"username": req.Username,
		"token":    tokenString,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
