package ui

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"ssg/ai"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

//go:embed static
var staticFS embed.FS

// Mount attaches all AI-manager routes onto r.
func Mount(r *gin.Engine, m *ai.Manager, defaultServer string) {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	r.GET("/", func(c *gin.Context) { serveIndex(c, sub, defaultServer) })
	r.StaticFS("/static", http.FS(sub))

	api := r.Group("/api")
	{
		api.GET("/agents", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"agents": m.List()})
		})
		api.POST("/agents", func(c *gin.Context) { handleAddAgent(c, m) })
		api.GET("/agents/:id", func(c *gin.Context) { handleAgentDetail(c, m) })
		api.DELETE("/agents/:id", func(c *gin.Context) { handleRemoveAgent(c, m) })
		api.POST("/agents/:id/pause", func(c *gin.Context) { handlePauseAgent(c, m, true) })
		api.POST("/agents/:id/resume", func(c *gin.Context) { handlePauseAgent(c, m, false) })
		api.GET("/games", handleListGames)
	}

	r.GET("/ws", func(c *gin.Context) { handleWS(c, m) })
}

func serveIndex(c *gin.Context, sub fs.FS, defaultServer string) {
	body, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "index missing")
		return
	}
	// Inject the default server URL so the dialog can prefill it without an extra round-trip.
	inject := `<script>window.AI_DEFAULT_SERVER = ` + jsonString(defaultServer) + `;</script></head>`
	out := strings.Replace(string(body), "</head>", inject, 1)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(out))
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

type addAgentRequest struct {
	Server   string `json:"server"`
	GameID   string `json:"game_id"`
	PlayerID uint   `json:"player_id"`
}

func handleAddAgent(c *gin.Context, m *ai.Manager) {
	var req addAgentRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := m.Add(req.Server, req.GameID, req.PlayerID, ai.SimplePolicy{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a.Summary())
}

func handleAgentDetail(c *gin.Context, m *ai.Manager) {
	id := c.Param("id")
	a, ok := m.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"summary":  a.Summary(),
		"log":      a.Log.Snapshot(),
		"snapshot": rawOrNil(a.Snapshot()),
	})
}

func rawOrNil(vg *ai.VisibleGame) interface{} {
	if vg == nil || len(vg.Raw) == 0 {
		return nil
	}
	return json.RawMessage(vg.Raw)
}

func handlePauseAgent(c *gin.Context, m *ai.Manager, paused bool) {
	id := c.Param("id")
	a, ok := m.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}
	a.SetPaused(paused)
	c.JSON(http.StatusOK, a.Summary())
}

func handleRemoveAgent(c *gin.Context, m *ai.Manager) {
	id := c.Param("id")
	if err := m.Remove(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleListGames(c *gin.Context) {
	server := c.Query("server")
	if server == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server query param is required"})
		return
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	body, err := ai.ListGames(ctx, server)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleWS(c *gin.Context, m *ai.Manager) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	events, cancel := m.Subscribe()
	defer cancel()

	ctx := c.Request.Context()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		}
	}
}

