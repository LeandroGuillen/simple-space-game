package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGameTestServer builds a server whose files stay inside temp directories.
func newGameTestServer(t *testing.T) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "custom-ship-designs.json"))
	s, err := NewServer()
	require.NoError(t, err)
	return s
}

func postNewGame(t *testing.T, s *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	out := httptest.NewRecorder()
	s.Handler().ServeHTTP(out, req)
	return out
}

// errorField returns the stable "error" field of a failed response.
func errorField(t *testing.T, out *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &body))
	value, ok := body["error"].(string)
	require.True(t, ok, "response has no string error field: %s", out.Body.String())
	require.NotEmpty(t, value)
	return value
}

func TestNewGameRejectsFixableSetupWithBadRequest(t *testing.T) {
	s := newGameTestServer(t)
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "invalid player color",
			body: `{"player_name":"Ada","num_stars":8,"players":1,"race":"human","player_color":"neon_pink"}`,
			want: `invalid player color "neon_pink"`,
		},
		{
			name: "more players than stars",
			body: `{"player_name":"Ada","num_stars":2,"players":5,"race":"human"}`,
			want: "players (5) cannot exceed stars (2)",
		},
		{
			name: "more players than allowed",
			body: `{"player_name":"Ada","num_stars":40,"players":99,"race":"human"}`,
			want: fmt.Sprintf("players must be between 1 and %d", model.MaxPlayers),
		},
		{
			name: "missing stars",
			body: `{}`,
			want: "players (1) cannot exceed stars (0)",
		},
		{
			name: "empty body",
			body: "",
			want: "players (1) cannot exceed stars (0)",
		},
		{
			name: "zero stars",
			body: `{"player_name":"Ada","num_stars":0,"players":1,"race":"human"}`,
			want: "players (1) cannot exceed stars (0)",
		},
		{
			name: "unknown race",
			body: `{"player_name":"Ada","num_stars":8,"players":1,"race":"klingon"}`,
			want: `unknown race "klingon"`,
		},
		{
			name: "unknown galaxy size",
			body: `{"player_name":"Ada","num_stars":8,"players":1,"race":"human","galaxy_size":"massive"}`,
			want: `unknown galaxy size "massive"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := postNewGame(t, s, tc.body)
			require.Equal(t, http.StatusBadRequest, out.Code, out.Body.String())
			assert.Contains(t, errorField(t, out), tc.want)
		})
	}
}

func TestNewGameAcceptsKnownGalaxySizes(t *testing.T) {
	s := newGameTestServer(t)
	for _, size := range []string{"", "tiny", "medium", "huge"} {
		t.Run(fmt.Sprintf("size %q", size), func(t *testing.T) {
			body := fmt.Sprintf(`{"player_name":"Ada","num_stars":8,"players":1,"race":"human","galaxy_size":%q}`, size)
			out := postNewGame(t, s, body)
			require.Equal(t, http.StatusOK, out.Code, out.Body.String())
		})
	}
}

func TestNewGameRejectsStarCountAboveCap(t *testing.T) {
	s := newGameTestServer(t)

	// 101 stars is one past the documented cap; 100000 is the request shape
	// that created 228 MB autosaves before the cap existed.
	for _, stars := range []uint{101, 100000} {
		out := postNewGame(t, s, fmt.Sprintf(`{"player_name":"Ada","num_stars":%d,"players":1,"race":"human"}`, stars))
		require.Equal(t, http.StatusBadRequest, out.Code, "num_stars=%d: %s", stars, out.Body.String())
		assert.Contains(t, errorField(t, out), "num_stars must be between 1 and 100")
	}
	assert.Empty(t, s.listGames(), "rejected requests must not leave a game behind")

	// Exactly the cap still works, matching the biggest galaxy the web client offers.
	atCap := postNewGame(t, s, `{"player_name":"Ada","num_stars":100,"players":1,"race":"human","galaxy_size":"huge"}`)
	require.Equal(t, http.StatusOK, atCap.Code, atCap.Body.String())
}

func TestNewGameRejectsMalformedBody(t *testing.T) {
	s := newGameTestServer(t)
	out := postNewGame(t, s, `{"num_stars":`)
	require.Equal(t, http.StatusBadRequest, out.Code, out.Body.String())
	assert.Contains(t, errorField(t, out), "invalid request body")
}

func TestNewGameRejectsWrongFieldTypes(t *testing.T) {
	s := newGameTestServer(t)
	out := postNewGame(t, s, `{"player_name":"Ada","num_stars":"eight","players":1,"race":"human"}`)
	require.Equal(t, http.StatusBadRequest, out.Code, out.Body.String())
	assert.Contains(t, errorField(t, out), "invalid request body")
}
