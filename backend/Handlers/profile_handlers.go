package Handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
)

const (
	maxBioLen       = 500
	maxAvatarURLLen = 2048

	// The PATCH body is tiny (bio + avatar URL), but json.Decoder
	// materializes the whole body before the length checks run, so cap
	// it to keep a huge upload from being read at all.
	maxProfileBodyBytes = 16 << 10
)

// GetMeHandler returns the authenticated user's own profile. Unlike
// the public view, this includes private fields such as email.
func (s *Server) GetMeHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int64)

	var me MeProfile
	query := `select id, username, email,
	        coalesce(bio, ''), coalesce(avatar_url, ''), created_at
	    from users where id = $1;`
	err := s.db.QueryRow(query, userID).
		Scan(&me.ID, &me.Username, &me.Email, &me.Bio, &me.AvatarURL, &me.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		fmt.Println("Database error: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(me)
}

// UpdateMeHandler updates the authenticated user's editable profile
// fields. Pointers in the request body mean: absent = leave unchanged,
// present = set (empty string clears the field).
func (s *Server) UpdateMeHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int64)

	r.Body = http.MaxBytesReader(w, r.Body, maxProfileBodyBytes)

	var req UpdateProfileRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Bio == nil && req.AvatarURL == nil {
		http.Error(w, "Nothing to update", http.StatusBadRequest)
		return
	}
	// Count runes, not bytes: "characters" in the message means
	// characters, and a byte check would reject CJK bios at ~1/3 length.
	if req.Bio != nil && utf8.RuneCountInString(*req.Bio) > maxBioLen {
		http.Error(w, fmt.Sprintf("Bio must be at most %d characters", maxBioLen), http.StatusBadRequest)
		return
	}
	if req.AvatarURL != nil && utf8.RuneCountInString(*req.AvatarURL) > maxAvatarURLLen {
		http.Error(w, fmt.Sprintf("Avatar URL must be at most %d characters", maxAvatarURLLen), http.StatusBadRequest)
		return
	}

	// Build the update dynamically so omitted fields are untouched
	// rather than overwritten with empty values.
	var sets []string
	args := []any{}
	if req.Bio != nil {
		sets = append(sets, fmt.Sprintf("bio = $%d", len(args)+1))
		args = append(args, *req.Bio)
	}
	if req.AvatarURL != nil {
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", len(args)+1))
		args = append(args, *req.AvatarURL)
	}
	args = append(args, userID)

	query := fmt.Sprintf(`update users set %s, updated_at = now()
	    where id = $%d
	    returning id, username, email, coalesce(bio, ''), coalesce(avatar_url, ''), created_at;`,
		strings.Join(sets, ", "), len(args))

	var me MeProfile
	err = s.db.QueryRow(query, args...).
		Scan(&me.ID, &me.Username, &me.Email, &me.Bio, &me.AvatarURL, &me.CreatedAt)
	if err != nil {
		// The token is validated without a DB lookup, so a deleted
		// account still yields valid tokens; zero rows here means the
		// user is gone, not that the server is broken.
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		fmt.Println("Database error: ", err)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(me)
}

// GetProfileHandler is the public view of a user by username — no
// email, no role, nothing private.
func (s *Server) GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var p Profile
	query := `select id, username, coalesce(bio, ''), coalesce(avatar_url, ''), created_at
	    from users where username = $1;`
	err := s.db.QueryRow(query, username).
		Scan(&p.ID, &p.Username, &p.Bio, &p.AvatarURL, &p.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		fmt.Println("Database error: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(p)
}
