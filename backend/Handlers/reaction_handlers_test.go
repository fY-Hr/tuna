package Handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveReactions routes through a real chi router so chi.URLParam
// extraction is exercised, matching how the profile handler tests do it.
func serveReactions(t *testing.T, s *Server, target string, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/api/waves/{id}/reactions", s.GetWaveReactionsHandler)
	r.Get("/api/waves/{id}/reactions/me", s.GetMyReactionHandler)

	var req *http.Request
	if userID != 0 {
		req = authedRequest(http.MethodGet, target, userID)
	} else {
		req = httptest.NewRequest(http.MethodGet, target, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- GetWaveReactionsHandler ---

func TestGetWaveReactionsHandler(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select exists\(select 1 from waves`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`filter \(where value = 1\)`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"up", "down"}).AddRow(int64(3), int64(1)))

	w := serveReactions(t, s, "/api/waves/5/reactions", 0)

	require.Equal(t, http.StatusOK, w.Code)
	var counts ReactionCounts
	require.NoError(t, json.NewDecoder(w.Body).Decode(&counts))
	assert.Equal(t, ReactionCounts{Up: 3, Down: 1}, counts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A wave with no reactions is fine: the aggregates coalesce to zero.
func TestGetWaveReactionsHandler_NoReactions(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select exists\(select 1 from waves`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`filter \(where value = 1\)`).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"up", "down"}).AddRow(int64(0), int64(0)))

	w := serveReactions(t, s, "/api/waves/5/reactions", 0)

	require.Equal(t, http.StatusOK, w.Code)
	var counts ReactionCounts
	require.NoError(t, json.NewDecoder(w.Body).Decode(&counts))
	assert.Equal(t, ReactionCounts{Up: 0, Down: 0}, counts)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetWaveReactionsHandler_WaveNotFound(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select exists\(select 1 from waves`).
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	w := serveReactions(t, s, "/api/waves/999/reactions", 0)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetWaveReactionsHandler_DBError(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select exists\(select 1 from waves`).
		WithArgs(int64(5)).
		WillReturnError(errors.New("connection refused"))

	w := serveReactions(t, s, "/api/waves/5/reactions", 0)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- GetMyReactionHandler ---

func TestGetMyReactionHandler_HasReaction(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select value from wave_reactions`).
		WithArgs(int64(1), int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(int16(-1)))

	w := serveReactions(t, s, "/api/waves/5/reactions/me", 1)

	require.Equal(t, http.StatusOK, w.Code)
	var my MyReaction
	require.NoError(t, json.NewDecoder(w.Body).Decode(&my))
	assert.Equal(t, int16(-1), my.Value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// No reaction yet is a normal state — value 0, not 404.
func TestGetMyReactionHandler_None(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select value from wave_reactions`).
		WithArgs(int64(1), int64(5)).
		WillReturnError(sql.ErrNoRows)

	w := serveReactions(t, s, "/api/waves/5/reactions/me", 1)

	require.Equal(t, http.StatusOK, w.Code)
	var my MyReaction
	require.NoError(t, json.NewDecoder(w.Body).Decode(&my))
	assert.Equal(t, int16(0), my.Value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMyReactionHandler_DBError(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`select value from wave_reactions`).
		WithArgs(int64(1), int64(5)).
		WillReturnError(errors.New("connection refused"))

	w := serveReactions(t, s, "/api/waves/5/reactions/me", 1)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
