package game

import (
	"sort"
	"strings"

	. "ssg/model"
)

// ShipDesignCatalog returns the current merged stock ship-design catalog (built-ins
// plus customs). The slice is immutable; concurrent catalog rebuilds publish a new
// snapshot without racing readers.
func (r *Resources) ShipDesignCatalog() []ShipDesign {
	if r == nil {
		return nil
	}
	coll := r.shipDesignCatalog.Load()
	if coll == nil {
		return nil
	}
	return coll.ShipDesigns
}

func (r *Resources) publishShipDesignCatalog(coll *ShipDesignCollection) {
	if r == nil || coll == nil {
		return
	}
	r.shipDesignCatalog.Store(coll)
}

func (r *Resources) loadedShipDesignCollection() *ShipDesignCollection {
	if r == nil {
		return nil
	}
	if coll := r.shipDesignCatalog.Load(); coll != nil {
		return coll
	}
	return r.ShipDesignCollection
}

// ApplyMergedShipDesignCatalog replaces the published catalog with built-ins plus sorted customs.
// It is copy-on-write: a fresh collection is built and published with a single
// atomic store, so readers keep a consistent view and never observe a half-written
// slice header.
func (r *ResourceManager) ApplyMergedShipDesignCatalog(custom []ShipDesign) {
	base := r.Resources.loadedShipDesignCollection()
	if r == nil || base == nil {
		return
	}
	sorted := append([]ShipDesign{}, custom...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	merged := append([]ShipDesign{}, r.BuiltinShipDesigns...)
	merged = append(merged, sorted...)
	r.Resources.publishShipDesignCatalog(&ShipDesignCollection{
		ShipDesignsDefs: base.ShipDesignsDefs,
		ShipDesigns:     merged,
	})
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
