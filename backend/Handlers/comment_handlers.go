package Handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) CreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)
	var newComment CreateCommentRequest
	err = json.NewDecoder(r.Body).Decode(&newComment)
	if err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(newComment.Content) == "" {
		writeError(w, "Comment content cant be empty", http.StatusBadRequest)
		return
	}
	newComment.Content = strings.TrimSpace(newComment.Content)

	waveExists, err := s.waveExists(waveID)
	if err != nil {
		fmt.Println("Database error: ", err)
		writeError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !waveExists {
		writeError(w, "Wave not found", http.StatusNotFound)
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
		fmt.Println("Database error: ", err)
		writeError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

func (s *Server) GetCommentsByWaveHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}

	waveExists, err := s.waveExists(waveID)
	if err != nil {
		fmt.Println("Database error: ", err)
		writeError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !waveExists {
		writeError(w, "Wave not found", http.StatusNotFound)
		return
	}

	query := `SELECT c.id, c.user_id, c.wave_id, c.content, c.created_at,
       u.username, COALESCE(u.avatar_url, '')
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.wave_id = $1
ORDER BY c.created_at ASC;`

	rows, err := s.db.Query(query, waveID)
	if err != nil {
		fmt.Println("Database error: ", err)
		writeError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	comments := []CommentResponse{}
	for rows.Next() {
		var a CommentResponse
		err = rows.Scan(&a.ID, &a.UserID, &a.WaveID, &a.Content, &a.CreatedAt, &a.Username, &a.AvatarURL)
		if err != nil {
			fmt.Println("Database error: ", err)
			writeError(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		comments = append(comments, a)
	}
	if err := rows.Err(); err != nil {
		fmt.Println("Database error: ", err)
		writeError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comments)
}

func (s *Server) DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	commentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, "Invalid Comment ID ", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)
	query := `DELETE FROM comments WHERE id = $1 AND user_id = $2;`
	queryExec, err := s.db.Exec(query, commentID, userID)
	if err != nil {
		writeError(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}
	rowsAffected, err := queryExec.RowsAffected()
	if err != nil {
		writeError(w, "Database error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		writeError(w, "Comment not found or unauthorized", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
