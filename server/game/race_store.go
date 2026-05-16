package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// RaceStore persists custom races to custom-races.json with atomic replace.
type RaceStore struct {
	path string
	mu   sync.Mutex
}

func NewRaceStore(path string) *RaceStore {
	return &RaceStore{path: path}
}

// Load reads custom races from disk. If the file is missing, it creates an empty collection file.
func (s *RaceStore) Load(resources *ResourceManager) ([]RaceDesign, error) {
	if s == nil {
		return nil, fmt.Errorf("race store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		if writeErr := s.writeFileLocked(resources, nil); writeErr != nil {
			return nil, writeErr
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var coll CustomRaceCollection
	if err := json.Unmarshal(data, &coll); err != nil {
		return nil, fmt.Errorf("parse custom races: %w", err)
	}

	seen := map[string]struct{}{}
	out := make([]RaceDesign, 0, len(coll.CustomRaces))
	for _, raw := range coll.CustomRaces {
		norm, err := SaveableCustomRace(resources, raw)
		if err != nil {
			return nil, fmt.Errorf("custom race %q: %w", raw.Name, err)
		}
		key := norm.Name
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("duplicate custom race name %q", key)
		}
		seen[key] = struct{}{}
		out = append(out, norm)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SaveAll replaces the entire custom race file after validating each design.
func (s *RaceStore) SaveAll(resources *ResourceManager, races []RaceDesign) error {
	if s == nil {
		return fmt.Errorf("race store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeFileLocked(resources, races)
}

func (s *RaceStore) writeFileLocked(resources *ResourceManager, races []RaceDesign) error {
	normalized := make([]RaceDesign, 0, len(races))
	seen := map[string]struct{}{}
	for _, raw := range races {
		norm, err := SaveableCustomRace(resources, raw)
		if err != nil {
			return err
		}
		if resources != nil && resources.IsBuiltInRaceName(norm.Name) {
			return fmt.Errorf("cannot persist custom race with built-in name %q", norm.Name)
		}
		if _, dup := seen[norm.Name]; dup {
			return fmt.Errorf("duplicate custom race name %q", norm.Name)
		}
		seen[norm.Name] = struct{}{}
		normalized = append(normalized, norm)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Name < normalized[j].Name })

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(CustomRaceCollection{CustomRaces: normalized}, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "custom-races-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
