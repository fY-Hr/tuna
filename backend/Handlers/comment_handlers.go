package Handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) CreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave id", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)
	var newComment CreateCommentRequest
	err = json.NewDecoder(r.Body).Decode(&newComment)
	if err != nil {
		http.Error(w, "Invalid body request!", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newComment.Content) == "" {
		http.Error(w, "Comment content cant be empty", http.StatusBadRequest)
		return
	}

	var waveExists bool

	query := `select exists(select 1 from waves where id = $1);`
	err = s.db.QueryRow(query, waveID).Scan(&waveExists)
	if err != nil {
		http.Error(w, "Database failed", http.StatusInternalServerError)
		return
	}
	if !waveExists {
		http.Error(w, "Wave doesn't exist", http.StatusNotFound)
		return
	}

	insertQuery := `INSERT INTO comments (user_id, wave_id, content)
	VALUES ($1, $2, $3)
	RETURNING id, created_at, updated_at;`

	var comment Comment
	comment.UserID = userID
	comment.WaveID = waveID
	comment.Content = newComment.Content

	err = s.db.QueryRow(insertQuery, userID, waveID, newComment.Content).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		http.Error(w, "Database Failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}
