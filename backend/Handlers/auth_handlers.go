package Handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// Usernames that collide with reserved URL segments. chi prefers static
// segments over params, so a user named "me" would make their public
// profile at /api/users/{username} permanently unreachable (the authed
// /api/users/me route always wins) — reserve these at registration.
var reservedUsernames = map[string]bool{
	"me": true,
}

// Access tokens are short-lived JWTs; refresh tokens are opaque,
// stored hashed, and single-use (rotated on every refresh).
const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// dbExec is satisfied by both *sql.DB and *sql.Tx, so token issuance
// can run standalone or inside the rotation transaction.
type dbExec interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// mintAccessToken signs a short-lived JWT for userID. It is pure
// crypto — no DB involved — so it can happen anywhere, even between
// transaction statements, and be rolled back for free.
func mintAccessToken(jwtSecret []byte, userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(accessTokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// newRefreshToken generates an opaque refresh token (32 random bytes,
// base64url) and stores only its SHA-256 hash, so a DB leak can't be
// replayed as a token.
func newRefreshToken(db dbExec, userID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	refreshToken := base64.RawURLEncoding.EncodeToString(raw)

	sum := sha256.Sum256([]byte(refreshToken))
	query := `insert into refresh_tokens (user_id, token_hash, expires_at)
	    values ($1, $2, $3);`
	_, err := db.Exec(query, userID, hex.EncodeToString(sum[:]), time.Now().Add(refreshTokenTTL))
	if err != nil {
		return "", err
	}
	return refreshToken, nil
}

// issueTokenPair mints a short-lived access JWT and an opaque refresh
// token. Used by login, where each piece failing independently is fine.
func (s *Server) issueTokenPair(userID int64) (accessToken, refreshToken string, err error) {
	accessToken, err = mintAccessToken(s.jwtSecret, userID)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = newRefreshToken(s.db, userID)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, "Invalid body request", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.Email) == "" {
		writeError(w, "Username, email, and password are required", http.StatusBadRequest)
		return
	}

	if reservedUsernames[strings.ToLower(req.Username)] {
		writeError(w, "That username is reserved", http.StatusConflict)
		return
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	var newID int64
	var createdAt time.Time

	query := `insert into users (username, email, password_hash)
    values ($1, $2, $3)
    returning id, created_at;`

	err = s.db.QueryRow(query, req.Username, req.Email, string(hashedBytes)).Scan(&newID, &createdAt)
	if err != nil {
		// Only a genuine unique-constraint violation means "taken";
		// everything else (DB down, etc.) is a server error, not 409.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, "Username or email already taken", http.StatusConflict)
			return
		}
		fmt.Println("Database error: ", err)
		writeError(w, "Registration failed", http.StatusInternalServerError)
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

// Login needs only username and password; using RegisterRequest here
// implied email was part of the login payload.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, "Invalid body request", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	var newID int64
	var storedHash []byte
	query := `select id, password_hash from users where username = $1`
	err = s.db.QueryRow(query, req.Username).Scan(&newID, &storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, "Failed to find user", http.StatusUnauthorized)
			return
		}
		fmt.Println("Database error: ", err)
		writeError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password))
	if err != nil {
		writeError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := s.issueTokenPair(newID)
	if err != nil {
		fmt.Println("Failed to issue token pair: ", err)
		writeError(w, "Login failed", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"id":            newID,
		"username":      req.Username,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(accessTokenTTL.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshHandler rotates a single-use refresh token: it validates the
// presented token, marks it used, and hands back a fresh access +
// refresh pair. Replaying an already-used token is treated as theft —
// every refresh token for that user is revoked.
func (s *Server) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.RefreshToken == "" {
		writeError(w, "Invalid body request", http.StatusBadRequest)
		return
	}

	sum := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := hex.EncodeToString(sum[:])

	// Single-use enforcement happens inside one transaction so a token
	// can never be consumed twice, even under concurrent refreshes.
	tx, err := s.db.Begin()
	if err != nil {
		writeError(w, "Refresh failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var userID int64
	var usedAt sql.NullTime
	var expiresAt time.Time
	query := `select user_id, used_at, expires_at from refresh_tokens
	    where token_hash = $1 for update;`
	err = tx.QueryRow(query, tokenHash).Scan(&userID, &usedAt, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}
		fmt.Println("Database error: ", err)
		writeError(w, "Refresh failed", http.StatusInternalServerError)
		return
	}

	// Reuse of an already-consumed token means it leaked: assume
	// compromise and revoke every refresh token for that user.
	if usedAt.Valid {
		if _, err := tx.Exec(`delete from refresh_tokens where user_id = $1;`, userID); err != nil {
			writeError(w, "Refresh failed", http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, "Refresh failed", http.StatusInternalServerError)
			return
		}
		writeError(w, "Refresh token reuse detected; all sessions revoked", http.StatusUnauthorized)
		return
	}

	if time.Now().After(expiresAt) {
		if _, err := tx.Exec(`delete from refresh_tokens where token_hash = $1;`, tokenHash); err != nil {
			writeError(w, "Refresh failed", http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, "Refresh failed", http.StatusInternalServerError)
			return
		}
		writeError(w, "Refresh token expired", http.StatusUnauthorized)
		return
	}

	// Consume the old token and mint its replacement inside the same
	// transaction. If anything here fails, the transaction rolls back:
	// the old token stays valid and the client can simply retry, with
	// no lockout and no false reuse alarm.
	if _, err := tx.Exec(`update refresh_tokens set used_at = now() where token_hash = $1;`, tokenHash); err != nil {
		writeError(w, "Refresh failed", http.StatusInternalServerError)
		return
	}
	newRefreshToken, err := newRefreshToken(tx, userID)
	if err != nil {
		fmt.Println("Failed to issue refresh token: ", err)
		writeError(w, "Refresh failed", http.StatusInternalServerError)
		return
	}
	newAccessToken, err := mintAccessToken(s.jwtSecret, userID)
	if err != nil {
		fmt.Println("Failed to issue access token: ", err)
		writeError(w, "Refresh failed", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, "Refresh failed", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    int(accessTokenTTL.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
