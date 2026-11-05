package game

import (
	"sort"
	"strings"

	. "ssg/model"
)

// ApplyMergedShipDesignCatalog replaces ShipDesignCollection.ShipDesigns with built-ins plus sorted customs.
// It is copy-on-write: a fresh collection is built and swapped in with a single
// pointer store, so readers that already hold the previous collection or its
// ShipDesigns slice keep a consistent view and never observe a half-written
// slice header.
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
	r.ShipDesignCollection = &ShipDesignCollection{
		ShipDesignsDefs: r.ShipDesignCollection.ShipDesignsDefs,
		ShipDesigns:     merged,
	}
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
