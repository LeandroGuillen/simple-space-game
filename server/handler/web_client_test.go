package server

import (
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHostedWebClientAndAPIRoutes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>SSG client</html>"), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "assets"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('ok')"), 0600))
	s, err := NewServerWithOptions(ServerOptions{WebDir: dir})
	require.NoError(t, err)
	for _, tc := range []struct {
		method, path string
		status       int
		contains     string
	}{
		{"GET", "/", 200, "SSG client"}, {"GET", "/galaxy", 200, "SSG client"}, {"GET", "/stars/1/planets/2", 200, "SSG client"}, {"GET", "/assets/app.js", 200, "console.log"}, {"GET", "/status", 200, "ok"}, {"GET", "/assets/missing.js", 404, ""}, {"GET", "/unknown-api", 404, ""}, {"POST", "/galaxy", 404, ""},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			require.Equal(t, tc.status, w.Code, w.Body.String())
			if tc.contains != "" {
				require.Contains(t, w.Body.String(), tc.contains)
			}
			if tc.path == "/galaxy" && tc.method == "GET" {
				require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
			}
		})
	}
}
func TestMissingWebClientFailsAtStartup(t *testing.T) {
	_, err := NewServerWithOptions(ServerOptions{WebDir: t.TempDir()})
	require.Error(t, err)
}
