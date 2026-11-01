package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"ssg/game"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MaxUploadedSaveBytes caps the raw body (or multipart file) accepted by /saves/upload.
const MaxUploadedSaveBytes = 20 << 20 // 20 MB

type SavedGameInfo struct {
	Name        string    `json:"name"`
	GameID      string    `json:"game_id"`
	CurrentTurn float64   `json:"current_turn"`
	NumStars    int       `json:"num_stars"`
	SavedAt     time.Time `json:"saved_at"`
	Size        int64     `json:"size"`
}

var saveNameRegexp = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeSaveName(name string) string {
	cleaned := saveNameRegexp.ReplaceAllString(strings.TrimSpace(name), "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		return ""
	}
	if len(cleaned) > 80 {
		cleaned = cleaned[:80]
	}
	return cleaned
}

func savePath(name string) (string, error) {
	clean := sanitizeSaveName(name)
	if clean == "" {
		return "", fmt.Errorf("invalid scenario name")
	}
	return filepath.Join(game.SavesDir(), clean+".json"), nil
}

func ListScenarios(c *gin.Context) {
	dir := game.SavesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"scenarios": []SavedGameInfo{}})
			return
		}
		log.Printf("cannot list scenarios: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list scenarios"})
		return
	}
	saves := make([]SavedGameInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		info, err := entry.Info()
		if err != nil {
			continue
		}
		header, _ := peekSavedGameHeader(filepath.Join(dir, entry.Name()))
		saves = append(saves, SavedGameInfo{
			Name:        name,
			GameID:      header.ID,
			CurrentTurn: header.CurrentTurn,
			NumStars:    header.NumStars,
			SavedAt:     info.ModTime(),
			Size:        info.Size(),
		})
	}
	sort.Slice(saves, func(i, j int) bool {
		return saves[i].SavedAt.After(saves[j].SavedAt)
	})
	c.JSON(http.StatusOK, gin.H{"scenarios": saves})
}

type savedGameHeader struct {
	ID          string  `json:"id"`
	CurrentTurn float64 `json:"current_turn"`
	NumStars    int     `json:"-"`
}

func peekSavedGameHeader(path string) (savedGameHeader, error) {
	var header struct {
		ID          string  `json:"id"`
		CurrentTurn float64 `json:"current_turn"`
		Galaxy      struct {
			Stars []json.RawMessage `json:"stars"`
		} `json:"galaxy"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return savedGameHeader{}, err
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return savedGameHeader{}, err
	}
	return savedGameHeader{
		ID:          header.ID,
		CurrentTurn: header.CurrentTurn,
		NumStars:    len(header.Galaxy.Stars),
	}, nil
}

func LoadScenario(c *gin.Context) {
	name := c.Param("name")
	path, err := savePath(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "scenario not found"})
			return
		}
		log.Printf("cannot read scenario: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read scenario"})
		return
	}

	s := c.MustGet("server").(*Server)

	var loaded game.Game
	if err := json.Unmarshal(data, &loaded); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot decode scenario: %v", err)})
		return
	}
	if loaded.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scenario file missing game id"})
		return
	}
	if err := game.ValidateLoadedGame(&loaded, s.ResourceManager); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	loaded.ID = uuid.New().String()
	game.HydrateLoadedGame(&loaded, s.ResourceManager)

	s.putGame(loaded.ID, &loaded)
	response := gin.H{"ok": true, "gameID": loaded.ID}
	if err := s.autosave(&loaded); err != nil {
		log.Printf("initial autosave failed for game %s: %v", loaded.ID, err)
		response["warning"] = "Game loaded, but autosave failed. Use Save now to retry."
	}
	c.JSON(http.StatusOK, response)
}

func LoadUploadedSave(c *gin.Context) {
	data, err := readUploadedGameData(c)
	if err != nil {
		if isUploadTooLarge(err) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("uploaded save exceeds %d byte limit", MaxUploadedSaveBytes),
			})
			return
		}
		log.Printf("cannot read upload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read upload"})
		return
	}

	s := c.MustGet("server").(*Server)

	var loaded game.Game
	if err := json.Unmarshal(data, &loaded); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot decode upload: %v", err)})
		return
	}
	if err := game.ValidateLoadedGame(&loaded, s.ResourceManager); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	loaded.ID = uuid.New().String()
	game.HydrateLoadedGame(&loaded, s.ResourceManager)

	s.putGame(loaded.ID, &loaded)
	response := gin.H{"ok": true, "gameID": loaded.ID}
	if err := s.autosave(&loaded); err != nil {
		log.Printf("initial autosave failed for game %s: %v", loaded.ID, err)
		response["warning"] = "Game loaded, but autosave failed. Use Save now to retry."
	}
	c.JSON(http.StatusOK, response)
}

func readUploadedGameData(c *gin.Context) ([]byte, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadedSaveBytes)
	if file, err := c.FormFile("file"); err == nil {
		if file.Size > MaxUploadedSaveBytes {
			return nil, &http.MaxBytesError{Limit: MaxUploadedSaveBytes}
		}
		open, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer open.Close()
		limited := http.MaxBytesReader(c.Writer, open, MaxUploadedSaveBytes)
		return io.ReadAll(limited)
	}
	return io.ReadAll(c.Request.Body)
}

func isUploadTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "http: request body too large") ||
		strings.Contains(msg, "request body too large")
}

// SaveGameNow persists a full server-side snapshot without returning private state.
func SaveGameNow(c *gin.Context) {
	var req GetGalaxyRequest
	if err := c.BindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if gameID != req.GameID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game id mismatch between URI and headers"})
		return
	}
	g, server, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}
	if _, err := g.GetVisibleGameForPlayer(playerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if server.autosaves == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server-side saves unavailable"})
		return
	}
	if err := server.autosave(g); err != nil {
		log.Printf("manual save failed for game %s: %v", gameID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save game"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
