package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// registerWebClient serves a Vite build without swallowing unknown API routes
// or exposing directory listings. WebDir is a trusted deployment directory.
func registerWebClient(router *gin.Engine, directory string) error {
	root, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("web client directory: %w", err)
	}
	index := filepath.Join(root, "index.html")
	info, err := os.Stat(index)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("web client index not found in %q", root)
	}
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path != "" && filepath.IsLocal(path) {
			file := filepath.Join(root, filepath.FromSlash(path))
			if info, err := os.Stat(file); err == nil && info.Mode().IsRegular() {
				http.ServeFile(c.Writer, c.Request, file)
				return
			}
		}
		if isClientRoute(c.Request.URL.Path) {
			c.Header("Cache-Control", "no-cache")
			http.ServeFile(c.Writer, c.Request, index)
			return
		}
		c.Status(http.StatusNotFound)
	})
	return nil
}

func isClientRoute(path string) bool {
	switch path {
	case "/", "/race-editor", "/ship-editor", "/setup", "/setup/races", "/setup/custom-race", "/setup/ship-editor", "/galaxy", "/colonies", "/fleets", "/technology", "/mutation":
		return true
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 2 && parts[0] == "battle" && parts[1] != "" || len(parts) == 4 && parts[0] == "stars" && parts[1] != "" && parts[2] == "planets" && parts[3] != ""
}
