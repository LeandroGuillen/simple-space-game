package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateGameAcceptsLegacyIgnoredSetupOptions(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)
	for _, body := range []string{
		`{"player_name":"Ada","num_stars":8,"players":1,"race":"human"}`,
		`{"player_name":"Ada","num_stars":8,"players":1,"race":"human","difficulty":"hard","galaxy_age":"mineral_rich","tech_level":"advanced"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/games", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		s.Handler().ServeHTTP(out, req)
		require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	}
}
