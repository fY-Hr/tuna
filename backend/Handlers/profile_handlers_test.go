package Handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return NewServer(db, []byte("test-secret")), mock
}

// authedRequest builds a request whose context carries userID, mimicking
// what the auth middleware does.
func authedRequest(method, target string, userID int64) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	return req.WithContext(context.WithValue(req.Context(), userIDKey, userID))
}

func meRows(id int64, username, email, bio, avatar string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "username", "email", "bio", "avatar_url", "created_at"}).
		AddRow(id, username, email, bio, avatar, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

func profileRows(id int64, username, bio, avatar string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "username", "bio", "avatar_url", "created_at"}).
		AddRow(id, username, bio, avatar, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// --- GetMeHandler ---

func TestGetMeHandler(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select id, username, email`).
		WithArgs(int64(1)).
		WillReturnRows(meRows(1, "aeon", "aeon@example.com", "hello", "http://a.png"))

	w := httptest.NewRecorder()
	s.GetMeHandler(w, authedRequest(http.MethodGet, "/api/users/me", 1))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var me MeProfile
	require.NoError(t, json.NewDecoder(w.Body).Decode(&me))
	assert.Equal(t, MeProfile{
		ID:        1,
		Username:  "aeon",
		Email:     "aeon@example.com",
		Bio:       "hello",
		AvatarURL: "http://a.png",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}, me)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMeHandler_NotFound(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select id, username, email`).
		WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	s.GetMeHandler(w, authedRequest(http.MethodGet, "/api/users/me", 42))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetMeHandler_DBError(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select id, username, email`).
		WithArgs(int64(1)).
		WillReturnError(errors.New("connection refused"))

	w := httptest.NewRecorder()
	s.GetMeHandler(w, authedRequest(http.MethodGet, "/api/users/me", 1))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- UpdateMeHandler ---

func TestUpdateMeHandler_Validation(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"invalid json", `{bad`, http.StatusBadRequest},
		{"nothing to update", `{"other": true}`, http.StatusBadRequest},
		{"bio too long", `{"bio": "` + strings.Repeat("a", maxBioLen+1) + `"}`, http.StatusBadRequest},
		{"avatar too long", `{"avatar_url": "` + strings.Repeat("a", maxAvatarURLLen+1) + `"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, mock := newTestServer(t)

			req := authedRequest(http.MethodPatch, "/api/users/me", 1)
			req.Body = io.NopCloser(strings.NewReader(tc.body))
			w := httptest.NewRecorder()
			s.UpdateMeHandler(w, req)

			assert.Equal(t, tc.want, w.Code)
			// Rejected before touching the database.
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateMeHandler_BioOnly(t *testing.T) {
	s, mock := newTestServer(t)

	// Only bio should be in the SET clause — avatar_url untouched, and
	// updated_at must keep getting bumped.
	mock.ExpectQuery(`update users set bio = \$1, updated_at = now\(\)`).
		WithArgs("new bio", int64(1)).
		WillReturnRows(meRows(1, "aeon", "aeon@example.com", "new bio", ""))

	req := authedRequest(http.MethodPatch, "/api/users/me", 1)
	req.Body = io.NopCloser(strings.NewReader(`{"bio": "new bio"}`))
	w := httptest.NewRecorder()
	s.UpdateMeHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var me MeProfile
	require.NoError(t, json.NewDecoder(w.Body).Decode(&me))
	assert.Equal(t, "new bio", me.Bio)
	assert.Equal(t, "aeon@example.com", me.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMeHandler_ClearBio(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`update users set bio = \$1, updated_at = now\(\)`).
		WithArgs("", int64(1)).
		WillReturnRows(meRows(1, "aeon", "aeon@example.com", "", ""))

	req := authedRequest(http.MethodPatch, "/api/users/me", 1)
	req.Body = io.NopCloser(strings.NewReader(`{"bio": ""}`))
	w := httptest.NewRecorder()
	s.UpdateMeHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var me MeProfile
	require.NoError(t, json.NewDecoder(w.Body).Decode(&me))
	assert.Empty(t, me.Bio)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The avatar_url branch and the both-fields branch exercise the dynamic
// SET builder's placeholder ordering (bio=$1, avatar_url=$2, id=$3) —
// regressions there previously passed the whole suite.
func TestUpdateMeHandler_AvatarOnly(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`update users set avatar_url = \$1, updated_at = now\(\)`).
		WithArgs("http://x.png", int64(1)).
		WillReturnRows(meRows(1, "aeon", "aeon@example.com", "old bio", "http://x.png"))

	req := authedRequest(http.MethodPatch, "/api/users/me", 1)
	req.Body = io.NopCloser(strings.NewReader(`{"avatar_url": "http://x.png"}`))
	w := httptest.NewRecorder()
	s.UpdateMeHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var me MeProfile
	require.NoError(t, json.NewDecoder(w.Body).Decode(&me))
	assert.Equal(t, "http://x.png", me.AvatarURL)
	assert.Equal(t, "old bio", me.Bio) // bio untouched
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMeHandler_BothFields(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`update users set bio = \$1, avatar_url = \$2, updated_at = now\(\)`).
		WithArgs("new bio", "http://x.png", int64(1)).
		WillReturnRows(meRows(1, "aeon", "aeon@example.com", "new bio", "http://x.png"))

	req := authedRequest(http.MethodPatch, "/api/users/me", 1)
	req.Body = io.NopCloser(strings.NewReader(`{"bio": "new bio", "avatar_url": "http://x.png"}`))
	w := httptest.NewRecorder()
	s.UpdateMeHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var me MeProfile
	require.NoError(t, json.NewDecoder(w.Body).Decode(&me))
	assert.Equal(t, "new bio", me.Bio)
	assert.Equal(t, "http://x.png", me.AvatarURL)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Limits are counted in runes: 250 CJK characters are ~750 bytes but
// only 250 characters, so they must be accepted.
func TestUpdateMeHandler_CJKBioUnderLimit(t *testing.T) {
	s, mock := newTestServer(t)

	bio := strings.Repeat("あ", 250)
	mock.ExpectQuery(`update users set bio = \$1`).
		WithArgs(bio, int64(1)).
		WillReturnRows(meRows(1, "aeon", "aeon@example.com", bio, ""))

	req := authedRequest(http.MethodPatch, "/api/users/me", 1)
	req.Body = io.NopCloser(strings.NewReader(`{"bio": "` + bio + `"}`))
	w := httptest.NewRecorder()
	s.UpdateMeHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMeHandler_DBError(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`update users set bio = \$1`).
		WithArgs("x", int64(1)).
		WillReturnError(errors.New("connection refused"))

	req := authedRequest(http.MethodPatch, "/api/users/me", 1)
	req.Body = io.NopCloser(strings.NewReader(`{"bio": "x"}`))
	w := httptest.NewRecorder()
	s.UpdateMeHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// A valid token for a since-deleted user yields zero updated rows;
// that is a 404, not a server error.
func TestUpdateMeHandler_NotFound(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`update users set bio = \$1`).
		WithArgs("x", int64(42)).
		WillReturnError(sql.ErrNoRows)

	req := authedRequest(http.MethodPatch, "/api/users/me", 42)
	req.Body = io.NopCloser(strings.NewReader(`{"bio": "x"}`))
	w := httptest.NewRecorder()
	s.UpdateMeHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- GetProfileHandler ---

func TestGetProfileHandler(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select id, username`).
		WithArgs("aeon").
		WillReturnRows(profileRows(1, "aeon", "hello", ""))

	// Route through a real chi router so URLParam extraction is tested.
	r := chi.NewRouter()
	r.Get("/api/users/{username}", s.GetProfileHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/users/aeon", nil))

	require.Equal(t, http.StatusOK, w.Code)

	var raw map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&raw))
	assert.Equal(t, "aeon", raw["username"])
	assert.Equal(t, "hello", raw["bio"])
	// Public view must not leak private fields.
	assert.NotContains(t, raw, "email")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProfileHandler_NotFound(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select id, username`).
		WithArgs("ghost").
		WillReturnError(sql.ErrNoRows)

	r := chi.NewRouter()
	r.Get("/api/users/{username}", s.GetProfileHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/users/ghost", nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProfileHandler_DBError(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select id, username`).
		WithArgs("aeon").
		WillReturnError(errors.New("connection refused"))

	r := chi.NewRouter()
	r.Get("/api/users/{username}", s.GetProfileHandler)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/users/aeon", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Called without a chi RouteContext, URLParam returns "" and the
// handler must reject rather than query with an empty username.
func TestGetProfileHandler_EmptyUsername(t *testing.T) {
	s, _ := newTestServer(t)

	w := httptest.NewRecorder()
	s.GetProfileHandler(w, httptest.NewRequest(http.MethodGet, "/api/users/", nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
