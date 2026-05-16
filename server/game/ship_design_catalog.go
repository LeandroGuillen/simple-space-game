package game

import (
	"sort"
	"strings"

	. "ssg/model"
)

// ApplyMergedShipDesignCatalog replaces ShipDesignCollection.ShipDesigns with built-ins plus sorted customs.
func (r *ResourceManager) ApplyMergedShipDesignCatalog(custom []ShipDesign) {
	if r == nil || r.ShipDesignCollection == nil {
		return
	}
	sorted := append([]ShipDesign{}, custom...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	merged := append([]ShipDesign{}, r.BuiltinShipDesigns...)
	merged = append(merged, sorted...)
	r.ShipDesignCollection.ShipDesigns = merged
}

// IsBuiltInShipDesignName reports whether name matches a built-in ship design (case-insensitive).
func (r *ResourceManager) IsBuiltInShipDesignName(name string) bool {
	if r == nil {
		return false
	}
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return false
	}
	for _, d := range r.BuiltinShipDesigns {
		if strings.ToLower(strings.TrimSpace(d.Name)) == key {
			return true
		}
	}
	return false
}
