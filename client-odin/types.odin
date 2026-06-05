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
	size:         int    `json:"size"`,
	population:   int    `json:"population"`,
	marines:      int    `json:"marines"`,
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
	width:  i64        `json:"width"`,
	height: i64        `json:"height"`,
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
	id:    int    `json:"id"`,
	name:  string `json:"name"`,
	color: string `json:"color"`,
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
}

Api_Game_Update :: struct {
	game_update: Api_Game `json:"game_update"`,
}

// Turn POST body. The server's Commands struct has many optional sub-fields;
// for the MVP we only send fleet_move. JSON marshal in core:encoding/json
// does not omit zero values by default, so we keep this struct minimal —
// only what we want to actually send.
Turn_Commands_Body :: struct {
	fleet_move: []Fleet_Move_Command `json:"fleet_move"`,
}

Fleet_Move_Command :: struct {
	fleet_id:            int   `json:"fleet_id"`,
	destination_star_id: int   `json:"destination_star_id"`,
	ship_indexes:        []int `json:"ship_indexes"`,
}

Turn_Request :: struct {
	game_id:  string             `json:"game_id"`,
	commands: Turn_Commands_Body `json:"commands"`,
}

Api_Turn_Response :: struct {
	ok:          bool           `json:"ok"`,
	advanced:    bool           `json:"advanced"`,
	turn_status: Api_Turn_Status `json:"turn_status"`,
}
