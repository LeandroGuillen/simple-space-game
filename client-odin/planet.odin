package main

// Phase 1 planet helpers: building catalog cache, planet lookups, and the
// buildable-buildings list. Pure reads over the current game snapshot plus a
// one-time cache of the (static) building catalog.

// name -> {cost, unique}. Cached once from the initial REST snapshot's
// resources so it survives WebSocket frames (which may omit `resources`).
building_catalog: map[string]Api_Building

cache_building_catalog :: proc(g: Api_Game) {
	if len(building_catalog) > 0 do return
	for b in g.resources.building_collection.buildings {
		building_catalog[b.name] = b
	}
}

building_info :: proc(name: string) -> (Api_Building, bool) {
	b, ok := building_catalog[name]
	return b, ok
}

// Assignable population units (the granularity move_population works in).
planet_units :: proc(p: Api_Planet) -> int {
	return p.population / 1000
}

// Find a planet by id across every star. The returned pointer is into the
// snapshot `g`, valid only for this frame.
find_planet :: proc(g: Api_Game, planet_id: int) -> (Api_Planet, bool) {
	for s in g.galaxy.stars {
		for p in s.planets {
			if p.id == planet_id do return p, true
		}
	}
	return {}, false
}

// Star that hosts a planet (for headers / navigation).
star_of_planet :: proc(g: Api_Game, planet_id: int) -> (Api_Star, bool) {
	for s in g.galaxy.stars {
		for p in s.planets {
			if p.id == planet_id do return s, true
		}
	}
	return {}, false
}

// Every planet owned by the player, with its star, ordered by star then planet.
owned_planets :: proc(g: Api_Game, player_id: int, allocator := context.allocator) -> []Api_Planet {
	out := make([dynamic]Api_Planet, 0, 16, allocator)
	for s in g.galaxy.stars {
		for p in s.planets {
			if p.owner == player_id do append(&out, p)
		}
	}
	return out[:]
}

player_available_buildings :: proc(g: Api_Game, player_id: int) -> []string {
	for p in g.players {
		if p.id == player_id do return p.available.buildings
	}
	return nil
}

// Buildings the player can queue on this planet: those they've unlocked, minus
// any unique building already standing here. (The server does the full
// canBuild validation; this just keeps the obvious cases out of the list.)
buildable_buildings :: proc(g: Api_Game, p: Api_Planet, player_id: int, allocator := context.allocator) -> []string {
	out := make([dynamic]string, 0, 16, allocator)
	for name in player_available_buildings(g, player_id) {
		if info, ok := building_info(name); ok && info.unique && planet_has_building(p, name) {
			continue
		}
		append(&out, name)
	}
	return out[:]
}

planet_has_building :: proc(p: Api_Planet, name: string) -> bool {
	for b in p.buildings {
		if b == name do return true
	}
	return false
}

// Population reallocation over a [farmers, workers, scientists] triple, keeping
// the total within `units`. Increasing a category first consumes any unassigned
// units, then pulls from the largest other category; decreasing frees a unit
// back to "unassigned".
pop_inc :: proc(alloc: ^[3]int, units, i: int) {
	if alloc[0] + alloc[1] + alloc[2] < units {
		alloc[i] += 1
		return
	}
	donor := -1
	for k in 0 ..< 3 {
		if k == i do continue
		if alloc[k] > 0 && (donor < 0 || alloc[k] > alloc[donor]) do donor = k
	}
	if donor >= 0 {
		alloc[donor] -= 1
		alloc[i] += 1
	}
}

pop_dec :: proc(alloc: ^[3]int, i: int) {
	if alloc[i] > 0 do alloc[i] -= 1
}

// --- Enum display names (mirror server/model/planet.go constants) -------

env_name :: proc(e: int) -> string {
	switch e {
	case 1:  return "Gas Giant"
	case 2:  return "Asteroids"
	case 3:  return "Toxic"
	case 4:  return "Radiated"
	case 5:  return "Barren"
	case 6:  return "Desert"
	case 7:  return "Arid"
	case 8:  return "Tundra"
	case 9:  return "Swamp"
	case 10: return "Ocean"
	case 11: return "Terran"
	case 12: return "Gaia"
	case:    return "?"
	}
}

size_name :: proc(s: int) -> string {
	switch s {
	case 1: return "Tiny"
	case 2: return "Small"
	case 3: return "Medium"
	case 4: return "Large"
	case 5: return "Huge"
	case:   return "?"
	}
}

mineral_name :: proc(m: int) -> string {
	switch m {
	case 1: return "Ultra Poor"
	case 2: return "Poor"
	case 3: return "Abundant"
	case 4: return "Rich"
	case 5: return "Ultra Rich"
	case:   return "?"
	}
}

gravity_name :: proc(gr: int) -> string {
	switch gr {
	case 1: return "Low-G"
	case 2: return "Normal-G"
	case 3: return "High-G"
	case:   return "?-G"
	}
}
