package server

import (
	"encoding/json"
	"fmt"
	"io"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	game.HydrateLoadedGame(&loaded, s.ResourceManager)

	s.putGame(loaded.ID, &loaded)
	c.JSON(http.StatusOK, gin.H{"ok": true, "gameID": loaded.ID})
}

func LoadUploadedSave(c *gin.Context) {
	data, err := readUploadedGameData(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot read upload: %v", err)})
		return
	}

	s := c.MustGet("server").(*Server)

	var loaded game.Game
	if err := json.Unmarshal(data, &loaded); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot decode upload: %v", err)})
		return
	}
	loaded.ID = uuid.New().String()
	game.HydrateLoadedGame(&loaded, s.ResourceManager)

	s.putGame(loaded.ID, &loaded)
	c.JSON(http.StatusOK, gin.H{"ok": true, "gameID": loaded.ID})
}

func readUploadedGameData(c *gin.Context) ([]byte, error) {
	if file, err := c.FormFile("file"); err == nil {
		open, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer open.Close()
		return io.ReadAll(open)
	}
	return io.ReadAll(c.Request.Body)
}
