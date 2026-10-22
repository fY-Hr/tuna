package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	//gorm
	_ "github.com/jackc/pgx/v5/stdlib"
)

var jwtSecret = []byte("super_secret_tuna_key_replace_later")

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Read Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. Strip "Bearer " and parse JWT
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 3. Extract userID
		userID := int64(claims["user_id"].(float64))

		// 4. Stash userID into the request's context backpack!
		ctx := context.WithValue(r.Context(), "userID", userID)

		// 5. Let the user walk through the door into the real handler!
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) getWavesByIDHandler(w http.ResponseWriter, r *http.Request) {
	var wave Wave
	var userID int64

	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}

	query := ` select id, user_id, content, created_at, is_archive
	 				from waves where id = $1`

	err = s.db.QueryRow(query, waveID).Scan(&wave.ID, &userID, &wave.Content, &wave.CreatedAt, &wave.IsArchive)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Wave not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wave)
}
func (s *Server) getWavesHandler(w http.ResponseWriter, r *http.Request) {
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
}

func (s *Server) createWavesHandler(w http.ResponseWriter, r *http.Request) {
	var newWave Wave
	err := json.NewDecoder(r.Body).Decode(&newWave)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("userID").(int64)

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
}

func (s *Server) deleteWaveHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value("userID").(int64)

	query := `DELETE FROM waves WHERE id = $1 AND user_id = $2;`
	res, err := s.db.Exec(query, waveID, userID)
	if err != nil {
		http.Error(w, "Failed to delete wave", http.StatusInternalServerError)
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Wave not found or unauthorized", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
