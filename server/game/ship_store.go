package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	. "ssg/model"
)

// ShipStore persists custom ship design templates to custom-ship-designs.json with atomic replace.
type ShipStore struct {
	path string
	mu   sync.Mutex
}

func NewShipStore(path string) *ShipStore {
	return &ShipStore{path: path}
}

// CustomShipDesignCollection is the custom-ship-designs.json envelope.
type CustomShipDesignCollection struct {
	CustomShipDesigns []ShipDesignsDef `json:"custom_ship_designs"`
}

// Load reads custom ship designs from disk. If the file is missing, it creates an empty collection file.
func (s *ShipStore) Load(resources *ResourceManager) ([]ShipDesign, error) {
	if s == nil {
		return nil, fmt.Errorf("ship store is nil")
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

	var coll CustomShipDesignCollection
	if err := json.Unmarshal(data, &coll); err != nil {
		return nil, fmt.Errorf("parse custom ship designs: %w", err)
	}

	seen := map[string]struct{}{}
	out := make([]ShipDesign, 0, len(coll.CustomShipDesigns))
	for _, raw := range coll.CustomShipDesigns {
		design, err := SaveableCustomShipDesign(resources, raw)
		if err != nil {
			return nil, fmt.Errorf("custom ship design %q: %w", raw.Name, err)
		}
		key := stringsLowerName(design.Name)
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("duplicate custom ship design name %q", design.Name)
		}
		seen[key] = struct{}{}
		out = append(out, design)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// SaveAll replaces the entire custom ship design file after validating each design.
func (s *ShipStore) SaveAll(resources *ResourceManager, designs []ShipDesign) error {
	if s == nil {
		return fmt.Errorf("ship store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeFileLocked(resources, designs)
}

func (s *ShipStore) writeFileLocked(resources *ResourceManager, designs []ShipDesign) error {
	normalized := make([]ShipDesign, 0, len(designs))
	seen := map[string]struct{}{}
	for _, raw := range designs {
		def := ShipDesignPartsDef(raw)
		design, err := SaveableCustomShipDesign(resources, def)
		if err != nil {
			return err
		}
		if resources != nil && resources.IsBuiltInShipDesignName(design.Name) {
			return fmt.Errorf("cannot persist custom ship design with built-in name %q", design.Name)
		}
		key := stringsLowerName(design.Name)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate custom ship design name %q", design.Name)
		}
		seen[key] = struct{}{}
		normalized = append(normalized, design)
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Name < normalized[j].Name
	})

	defs := make([]ShipDesignsDef, 0, len(normalized))
	for _, d := range normalized {
		defs = append(defs, ShipDesignPartsDef(d))
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(CustomShipDesignCollection{CustomShipDesigns: defs}, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "custom-ship-designs-*.json")
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

func stringsLowerName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
