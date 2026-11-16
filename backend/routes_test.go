package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fY-Hr/tuna/backend/Handlers"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("test-secret")

// newRoutedServer builds the real route table backed by a sqlmock DB,
// plus the mock for setting query expectations.
func newRoutedServer(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return setupRoutes(Handlers.NewServer(db, testSecret)), mock
}

func bearerToken(userID int) string {
	claims := jwt.MapClaims{
		"user_id": float64(userID), // JSON numbers decode as float64
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testSecret)
	if err != nil {
		panic(err)
	}
	return token
}

// /api/users/me must match the static route inside the auth group, not
// the public /api/users/{username} route — otherwise the response would
// be the public profile and 401 would never fire for it.
func TestRouteUsersMePrefersStatic(t *testing.T) {
	h, mock := newRoutedServer(t)

	mock.ExpectQuery(`select id, username, email`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "bio", "avatar_url", "created_at"}).
			AddRow(1, "aeon", "aeon@example.com", "", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+bearerToken(1))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"email"`, "static route won: /me returned the owner view")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRouteUsersUsernamePublic(t *testing.T) {
	h, mock := newRoutedServer(t)

	mock.ExpectQuery(`select id, username`).
		WithArgs("aeon").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "bio", "avatar_url", "created_at"}).
			AddRow(1, "aeon", "", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/users/aeon", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), `"email"`, "public view must not leak email")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// /api/waves/{id}/reactions/me must live inside the auth group (it
// needs the caller's identity), while the counts endpoint stays public.
func TestRouteReactions(t *testing.T) {
	t.Run("counts are public", func(t *testing.T) {
		h, mock := newRoutedServer(t)

		mock.ExpectQuery(`select exists\(select 1 from waves`).
			WithArgs(int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`filter \(where value = 1\)`).
			WithArgs(int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"up", "down"}).AddRow(int64(2), int64(0)))

		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/waves/5/reactions", nil))

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"up":2`)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("reactions/me requires auth", func(t *testing.T) {
		h, _ := newRoutedServer(t)

		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/waves/5/reactions/me", nil))

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("reactions/me returns the caller's reaction", func(t *testing.T) {
		h, mock := newRoutedServer(t)

		mock.ExpectQuery(`select value from wave_reactions`).
			WithArgs(int64(1), int64(5)).
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(int16(1)))

		req := httptest.NewRequest(http.MethodGet, "/api/waves/5/reactions/me", nil)
		req.Header.Set("Authorization", "Bearer "+bearerToken(1))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"value":1`)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// The static /api/users/me route lives inside the auth group: hitting it
// without a token must 401 rather than fall through to the public
// param route.
func TestRouteUsersMeRequiresAuth(t *testing.T) {
	h, _ := newRoutedServer(t)

	for _, tc := range []struct {
		name   string
		method string
	}{
		{"get", http.MethodGet},
		{"patch", http.MethodPatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(tc.method, "/api/users/me", nil))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
