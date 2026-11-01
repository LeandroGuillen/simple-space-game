package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ssg/game"
)

const autosaveRetention = 5

type autosaveStore struct {
	dir string
	mu  sync.Mutex
}

func autosaveGameDir(root, id string) string {
	sum := sha256.Sum256([]byte(id))
	return filepath.Join(root, hex.EncodeToString(sum[:]))
}

// Save serializes both snapshotting and writing, preventing concurrent turns
// from overwriting a newer snapshot with an older one.
func (s *autosaveStore) Save(g *game.Game) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := g.MarshalSave()
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	dir := autosaveGameDir(s.dir, g.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, fmt.Sprintf("%020d-*.tmp", time.Now().UnixNano()))
	if err != nil {
		return err
	}
	tempPath := tmp.Name()
	defer os.Remove(tempPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	finalPath := strings.TrimSuffix(tempPath, ".tmp") + ".json"
	if err := os.Rename(tempPath, finalPath); err != nil {
		return err
	}
	// Sync the directory so the renamed file is durable across a system crash.
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = d.Sync()
	closeErr := d.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	files, err := autosaveFiles(dir)
	if err != nil {
		return err
	}
	for _, name := range files[minimum(autosaveRetention, len(files)):] {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// autosaveFiles returns newest-first completed snapshots, excluding temp files.
func autosaveFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	return names, nil
}

func (s *autosaveStore) Load(resources *game.ResourceManager) ([]*game.Game, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var games []*game.Game
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(s.dir, entry.Name())
		files, err := autosaveFiles(dir)
		if err != nil {
			log.Printf("autosave: cannot list %s: %v", entry.Name(), err)
			continue
		}
		for _, name := range files {
			path := filepath.Join(dir, name)
			f, err := os.Open(path)
			if err != nil {
				log.Printf("autosave: cannot open %s: %v", name, err)
				continue
			}
			var g game.Game
			decoder := json.NewDecoder(f)
			err = decoder.Decode(&g)
			if err == nil {
				var extra json.RawMessage
				if trailingErr := decoder.Decode(&extra); trailingErr != io.EOF {
					if trailingErr == nil {
						err = fmt.Errorf("unexpected trailing JSON")
					} else {
						err = trailingErr
					}
				}
			}
			f.Close()
			if err != nil {
				log.Printf("autosave: invalid %s: %v", name, err)
				continue
			}
			if g.ID == "" || autosaveGameDir(s.dir, g.ID) != dir {
				log.Printf("autosave: mismatched game ID in %s", name)
				continue
			}
			if err := game.ValidateLoadedGame(&g, resources); err != nil {
				log.Printf("autosave: invalid game in %s: %v", name, err)
				continue
			}
			game.HydrateLoadedGame(&g, resources)
			g.ResetTurnStatusAfterRestart()
			games = append(games, &g)
			break
		}
	}
	return games, nil
}

func (s *Server) autosave(g *game.Game) error {
	if s.autosaves == nil {
		return nil
	}
	return s.autosaves.Save(g)
}
