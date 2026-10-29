package main

import "core:encoding/json"
import "core:testing"

@(test)
test_invasion_outcome_label_cancelled :: proc(t: ^testing.T) {
	r := Api_Invasion_Report{cancelled = true, success = false, planet_captured = false}
	testing.expect_value(t, invasion_outcome_label(r), "cancelled (orbit lost)")
}

@(test)
test_invasion_outcome_label_captured :: proc(t: ^testing.T) {
	r := Api_Invasion_Report{planet_captured = true, success = true}
	testing.expect_value(t, invasion_outcome_label(r), "captured")
}

@(test)
test_invasion_outcome_label_won :: proc(t: ^testing.T) {
	r := Api_Invasion_Report{success = true, planet_captured = false}
	testing.expect_value(t, invasion_outcome_label(r), "won")
}

@(test)
test_invasion_outcome_label_repelled :: proc(t: ^testing.T) {
	r := Api_Invasion_Report{success = false, planet_captured = false}
	testing.expect_value(t, invasion_outcome_label(r), "repelled")
}

// Old draw_invasion_reports logic (pre-fix): cancelled reports were labelled
// "repelled". Kept here so a regression that reverts the label check fails
// this assertion against the fixed invasion_outcome_label.
@(test)
test_old_logic_mislabelled_cancelled_as_repelled :: proc(t: ^testing.T) {
	r := Api_Invasion_Report{cancelled = true, success = false, planet_captured = false}
	old := r.planet_captured ? "captured" : (r.success ? "won" : "repelled")
	testing.expect_value(t, old, "repelled")
	testing.expect(t, invasion_outcome_label(r) != old)
	testing.expect_value(t, invasion_outcome_label(r), "cancelled (orbit lost)")
}

@(test)
test_parse_invasion_report_cancelled_true :: proc(t: ^testing.T) {
	payload := "{\"id\":7,\"turn\":3,\"planet_id\":11,\"planet_name\":\"Grey Harbor\",\"attacker_owner_id\":1,\"defender_owner_id\":2,\"success\":false,\"planet_captured\":false,\"cancelled\":true,\"initial_attacker\":{\"owner_id\":1,\"marines\":[],\"armored\":[]},\"initial_defender\":{\"owner_id\":2,\"marines\":[],\"armored\":[]},\"surviving_units\":{\"owner_id\":0,\"marines\":[],\"armored\":[]}}"
	r: Api_Invasion_Report
	err := json.unmarshal(transmute([]u8)payload, &r, allocator = context.temp_allocator)
	testing.expect(t, err == nil, "unmarshal cancelled:true report")
	testing.expect(t, r.cancelled)
	testing.expect_value(t, invasion_outcome_label(r), "cancelled (orbit lost)")
}

@(test)
test_parse_invasion_report_without_cancelled_field :: proc(t: ^testing.T) {
	payload := "{\"id\":8,\"turn\":4,\"planet_id\":12,\"planet_name\":\"Old Save\",\"attacker_owner_id\":1,\"defender_owner_id\":2,\"success\":false,\"planet_captured\":false,\"initial_attacker\":{\"owner_id\":1,\"marines\":[],\"armored\":[]},\"initial_defender\":{\"owner_id\":2,\"marines\":[],\"armored\":[]},\"surviving_units\":{\"owner_id\":0,\"marines\":[],\"armored\":[]}}"
	r: Api_Invasion_Report
	err := json.unmarshal(transmute([]u8)payload, &r, allocator = context.temp_allocator)
	testing.expect(t, err == nil, "unmarshal report without cancelled")
	testing.expect(t, !r.cancelled)
	testing.expect_value(t, invasion_outcome_label(r), "repelled")
}
