package main

// JSON-mapped structs that mirror the server's wire format. These match
// the TypeScript types in client-solid/src/lib/types/api.ts (which is
// the most readable reference); fields not needed for the MVP are omitted.
//
// json:"name" tags map struct fields onto JSON keys. core:encoding/json
// silently ignores unknown JSON fields, so partial coverage is fine.

Api_Status :: struct {
	ok: bool `json:"ok"`,
}

Api_Game_List_Player :: struct {
	id:        int    `json:"id"`,
	name:      string `json:"name"`,
	race_name: string `json:"race_name"`,
	color:     string `json:"color"`,
	num_stars: int    `json:"num_stars"`,
}

Api_Game_List_Item :: struct {
	game_id:      string                 `json:"game_id"`,
	num_stars:    int                    `json:"num_stars"`,
	current_turn: int                    `json:"current_turn"`,
	players:      []Api_Game_List_Player `json:"players"`,
}

Api_Game_List_Response :: struct {
	game_list: []Api_Game_List_Item `json:"game_list"`,
}

Api_Planet :: struct {
	id:           int    `json:"id"`,
	name:         string `json:"name"`,
	owner:        int    `json:"owner"`,
	parent_star:  int    `json:"parent_star"`,

	mineral_class: int `json:"mineral_class"`,
	size:          int `json:"size"`,
	gravity:       int `json:"gravity"`,
	environment:   int `json:"environment"`,

	population:           int `json:"population"`,
	max_population:       int `json:"max_population"`,
	last_turn_pop_growth: int `json:"last_turn_pop_growth"`,
	marines:              int `json:"marines"`,

	food:     Api_Food_Output     `json:"food"`,
	industry: Api_Industry_Output `json:"industry"`,
	science:  Api_Science_Output  `json:"science"`,
	income:   Api_Income_Output   `json:"income"`,

	current_project: string   `json:"current_project"`,
	build_queue:     []string `json:"build_queue"`,
	buildings:       []string `json:"buildings"`,
	build_bucket:    int      `json:"build_bucket"`,
	industry_stack:  int      `json:"industry_stack"`,
	is_building:     bool     `json:"is_building"`,
	autobuild:       bool     `json:"autobuild"`,
}

Api_Food_Output :: struct {
	farmers:               int `json:"farmers"`,
	produced_per_farmer:   int `json:"produced_per_farmer"`,
	produced_per_building: int `json:"produced_per_building"`,
}

Api_Industry_Output :: struct {
	workers:               int `json:"workers"`,
	produced_per_worker:   int `json:"produced_per_worker"`,
	produced_per_building: int `json:"produced_per_building"`,
}

Api_Science_Output :: struct {
	scientists:            int `json:"scientists"`,
	produced_per_scientist:int `json:"produced_per_scientist"`,
	produced_per_building: int `json:"produced_per_building"`,
}

Api_Income_Output :: struct {
	taxes:       int `json:"taxes"`,
	buildings:   int `json:"buildings"`,
	trade_goods: int `json:"trade_goods"`,
	maintenance: int `json:"maintenance"`,
	total:       int `json:"total"`,
}

Api_Star :: struct {
	id:      int          `json:"id"`,
	name:    string       `json:"name"`,
	x:       f64          `json:"x"`,
	y:       f64          `json:"y"`,
	color:   int          `json:"color"`,
	planets: []Api_Planet `json:"planets"`,
}

Api_Galaxy :: struct {
	stars:  []Api_Star `json:"stars"`,
	width:  f64        `json:"width"`,
	height: f64        `json:"height"`,
}

Api_Fleet_Ship :: struct {
	type: string `json:"type"`,
	name: string `json:"name"`,
}

Api_Fleet :: struct {
	id:                   int              `json:"id"`,
	owner:                int              `json:"owner"`,
	star_id:              int              `json:"star_id"`,
	x:                    f64              `json:"x"`,
	y:                    f64              `json:"y"`,
	origin_star_id:       int              `json:"origin_star_id"`,
	destination_star_id:  int              `json:"destination_star_id"`,
	total_turns:          int              `json:"total_turns"`,
	turns_remaining:      int              `json:"turns_remaining"`,
	ships:                []Api_Fleet_Ship `json:"ships"`,
}

Api_Player :: struct {
	id:        int           `json:"id"`,
	name:      string        `json:"name"`,
	color:     string        `json:"color"`,
	treasury:  f64           `json:"treasury"`,
	available: Api_Available `json:"available"`,
}

// Per-player unlocked content. Only `buildings` is used so far (the list of
// buildings the player may construct).
Api_Available :: struct {
	buildings: []string `json:"buildings"`,
}

// Subset of the game's `resources` payload: just the building catalog, which
// supplies costs/uniqueness for the construction UI.
Api_Resources :: struct {
	building_collection: Api_Building_Collection `json:"building_collection"`,
}

Api_Building_Collection :: struct {
	buildings: []Api_Building `json:"buildings"`,
}

Api_Building :: struct {
	name:   string `json:"name"`,
	cost:   int    `json:"cost"`,
	unique: bool   `json:"unique"`,
}

Api_Turn_Status :: struct {
	mode:               string `json:"mode"`,
	ready_player_ids:   []int  `json:"ready_player_ids"`,
	waiting_player_ids: []int  `json:"waiting_player_ids"`,
	required_player_ids:[]int  `json:"required_player_ids"`,
}

Api_Game :: struct {
	id:           string         `json:"id"`,
	current_turn: int            `json:"current_turn"`,
	galaxy:       Api_Galaxy     `json:"galaxy"`,
	players:      []Api_Player   `json:"players"`,
	fleets:       []Api_Fleet    `json:"fleets"`,
	turn_status:  Api_Turn_Status `json:"turn_status"`,
	resources:    Api_Resources  `json:"resources"`,
}

Api_Game_Update :: struct {
	game_update: Api_Game `json:"game_update"`,
}

// The /turn command sub-structs below mirror the server's `model.Commands`
// (server/model/commands.go) and the web client's TurnCommands
// (client-solid/src/lib/types/commands.ts). The commands *object* itself is
// assembled by hand in commands.odin (build_turn_body) rather than via a
// single struct marshal, because Odin's json.marshal cannot serialise
// non-nil pointer fields — so optional singular commands can't be modelled
// as pointers. Manual assembly emits only the command kinds actually queued
// and lets us keep the server's exact (and deliberately inconsistent) keys:
// `movePopulation`/`selectResearch` are camelCase, the rest snake_case.

Build_Command :: struct {
	planet_id:   int      `json:"planetId"`,
	building:    string   `json:"building"`,
	build_queue: []string `json:"build_queue,omitempty"`,
}

Move_Population_Command :: struct {
	planet_id:  int `json:"planetId"`,
	farmers:    int `json:"farmers"`,
	workers:    int `json:"workers"`,
	scientists: int `json:"scientists"`,
}

Transfer_Population_Command :: struct {
	source_planet_id:      int `json:"source_planet_id"`,
	destination_planet_id: int `json:"destination_planet_id"`,
	population_units:      int `json:"population_units"`,
}

Select_Research_Command :: struct {
	tech_name: string `json:"tech_name"`,
}

Colonize_Command :: struct {
	planet_id: int `json:"planet_id"`,
	fleet_id:  int `json:"fleet_id,omitempty"`,
}

Artificial_Planet_Command :: struct {
	source_planet_id: int `json:"source_planet_id,omitempty"`,
	target_planet_id: int `json:"target_planet_id"`,
}

Invade_Command :: struct {
	fleet_id:         int   `json:"fleet_id"`,
	target_planet_id: int   `json:"target_planet_id"`,
	ship_indexes:     []int `json:"ship_indexes,omitempty"`,
}

Buy_Production_Command :: struct {
	planet_id: int `json:"planetId"`,
}

Mutate_Race_Command :: struct {
	trait_ids: []string `json:"trait_ids"`,
}

// ship_indexes uses omitempty so an empty slice is omitted entirely: the
// server treats an omitted field as "move the whole fleet", but an empty
// array (`[]`) as "no ships selected" and rejects it. So: empty => whole
// fleet, populated => exactly those ship indexes.
Fleet_Move_Command :: struct {
	fleet_id:            int   `json:"fleet_id"`,
	destination_star_id: int   `json:"destination_star_id"`,
	ship_indexes:        []int `json:"ship_indexes,omitempty"`,
}

// POST /games body. All fields omitempty; the server fills sensible
// defaults for anything left out (see handler.NewGameRequest).
New_Game_Request :: struct {
	num_stars:    int    `json:"num_stars,omitempty"`,
	player_name:  string `json:"player_name,omitempty"`,
	player_color: string `json:"player_color,omitempty"`,
	difficulty:   string `json:"difficulty,omitempty"`,
	galaxy_size:  string `json:"galaxy_size,omitempty"`,
	galaxy_age:   string `json:"galaxy_age,omitempty"`,
	players:      int    `json:"players,omitempty"`,
	tech_level:   string `json:"tech_level,omitempty"`,
	race:         string `json:"race,omitempty"`,
}

// POST /games response: {"ok": true, "gameID": "..."}.
New_Game_Response :: struct {
	ok:      bool   `json:"ok"`,
	game_id: string `json:"gameID"`,
}

Api_Turn_Response :: struct {
	ok:          bool           `json:"ok"`,
	advanced:    bool           `json:"advanced"`,
	turn_status: Api_Turn_Status `json:"turn_status"`,
}
