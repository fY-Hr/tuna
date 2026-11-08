package Handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	// gorm
	_ "github.com/jackc/pgx/v5/stdlib"
)

func (s *Server) GetWavesByIDHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) GetWavesHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) CreateWavesHandler(w http.ResponseWriter, r *http.Request) {
	var newWave Wave
	err := json.NewDecoder(r.Body).Decode(&newWave)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(userIDKey).(int64)

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

func (s *Server) DeleteWaveHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)

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

func (s *Server) ReactWaveHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)

	var req Reaction
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Value != 1 && req.Value != -1 && req.Value != 0 {
		http.Error(w, "Value must either be 1, -1, or 0", http.StatusBadRequest)
		return
	}

	// verify the wave exists before upserting, so FK violations don't become 500s
	var waveExists bool
	err = s.db.QueryRow(`select exists(select 1 from waves where id = $1)`, waveID).Scan(&waveExists)
	if err != nil {
		fmt.Println("Database Error: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !waveExists {
		http.Error(w, "Wave not found", http.StatusNotFound)
		return
	}

	if req.Value == 0 {
		query := ` delete from wave_reactions where user_id = $1 and wave_id = $2;`
		_, err = s.db.Exec(query, userID, waveID)
	} else {
		query := ` Insert into wave_reactions (user_id, wave_id, value) 
 values ($1, $2, $3)
 on conflict (user_id, wave_id) 
     do update set value = excluded.value, updated_at = now();`
		_, err = s.db.Exec(query, userID, waveID, req.Value)
	}
	if err != nil {
		http.Error(w, "Failed to react wave", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"message": "Reaction updated",
		"value":   req.Value,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetWaveReactionsHandler returns a wave's reaction counts. Public:
// the counts are aggregates and reveal nothing about individual users.
func (s *Server) GetWaveReactionsHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}

	var waveExists bool
	err = s.db.QueryRow(`select exists(select 1 from waves where id = $1)`, waveID).Scan(&waveExists)
	if err != nil {
		fmt.Println("Database error: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !waveExists {
		http.Error(w, "Wave not found", http.StatusNotFound)
		return
	}

	var counts ReactionCounts
	query := `select
	        coalesce(sum(value) filter (where value = 1), 0),
	        coalesce(sum(value) filter (where value = -1), 0)
	    from wave_reactions where wave_id = $1;`
	err = s.db.QueryRow(query, waveID).Scan(&counts.Up, &counts.Down)
	if err != nil {
		fmt.Println("Database error: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(counts)
}

// GetMyReactionHandler returns the calling user's own reaction on a
// wave. Having none is a normal state, not a 404 — value 0 means "no
// reaction", matching what POST /react accepts to clear one.
func (s *Server) GetMyReactionHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave ID", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)

	var my MyReaction
	query := `select value from wave_reactions where user_id = $1 and wave_id = $2;`
	err = s.db.QueryRow(query, userID, waveID).Scan(&my.Value)
	if err != nil {
		if err == sql.ErrNoRows {
			my.Value = 0
		} else {
			fmt.Println("Database error: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(my)
}

func (s *Server) ArchiveWaveHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	waveID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid wave id", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)
	query := `UPDATE waves SET is_archive = true, updated_at = NOW() WHERE id = $1 AND user_id = $2;`
	res, err := s.db.Exec(query, waveID, userID)
	if err != nil {
		http.Error(w, "Invalid user", http.StatusInternalServerError)
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "Invalid request", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Wave not found or unauthorized", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]any{
		"message": "Wave archived successfully",
		"id":      waveID,
	}
	json.NewEncoder(w).Encode(response)
}
