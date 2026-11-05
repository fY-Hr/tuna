package Handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Registering "me" in any case would shadow the public
// /api/users/{username} route with the authed /api/users/me routes
// (chi prefers static segments), making that user's public profile
// permanently unreachable — so it must be rejected before any DB work.
func TestRegisterHandler_ReservesMe(t *testing.T) {
	for _, username := range []string{"me", "Me", "ME"} {
		t.Run(username, func(t *testing.T) {
			s, mock := newTestServer(t)

			body := `{"username": "` + username + `", "email": "x@example.com", "password": "hunter2"}`
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
			w := httptest.NewRecorder()
			s.RegisterHandler(w, req)

			require.Equal(t, http.StatusConflict, w.Code)
			// Rejected before bcrypt or any database call.
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
