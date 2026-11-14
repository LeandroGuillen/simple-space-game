package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The standalone editor has no player, so its parts list names the research
// each weapon needs instead of filtering by it.
func TestStandaloneShipPartsNameEachWeaponsRequiredTech(t *testing.T) {
	s, g := designBuildServer(t)
	w := designBuildRequest(t, s, g, http.MethodGet, "/resources/shipparts", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Beams []struct {
			Name         string `json:"name"`
			MaxDamage    int64  `json:"maxDamage"`
			RequiredTech string `json:"required_tech"`
		} `json:"beams"`
		Missiles []struct {
			Name         string `json:"name"`
			RequiredTech string `json:"required_tech"`
		} `json:"missiles"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	beams := map[string]string{}
	for _, b := range body.Beams {
		beams[b.Name] = b.RequiredTech
		if b.MaxDamage > 0 {
			assert.NotEmpty(t, b.RequiredTech, "beam %q", b.Name)
		}
	}
	missiles := map[string]string{}
	for _, m := range body.Missiles {
		missiles[m.Name] = m.RequiredTech
		assert.NotEmpty(t, m.RequiredTech, "missile %q", m.Name)
	}
	assert.Equal(t, "Physics", beams["Laser"])
	assert.Equal(t, "Multi-Phased Physics", beams["Phasors"])
	assert.Equal(t, "Chemistry", missiles["Nuclear Missile"])
	assert.Equal(t, "Molecular Manipulation", missiles["Zeon Missile"])

	// Designs never carry the editor-only annotation.
	require.Equal(t, http.StatusOK, postCustomShipDesign(t, s, armedCustomShipPayload("Lancer", "beam", "Laser")).Code)
	saved, err := json.Marshal(s.CustomShipDesigns["Lancer"])
	require.NoError(t, err)
	assert.Contains(t, string(saved), `"Laser"`)
	assert.NotContains(t, string(saved), "required_tech")
}

func armedCustomShipPayload(name, slot, weapon string) map[string]any {
	payload := sampleCustomShipPayload(name)
	payload[slot] = weapon
	return payload
}

func TestCatalogTemplateWithUnresearchedWeaponIsRejectedInGame(t *testing.T) {
	for _, tc := range []struct{ slot, weapon, want string }{
		{"missile", "Zeon Missile", `missile "Zeon Missile" has not been researched`},
		{"beam", "Phasors", `beam "Phasors" has not been researched`},
	} {
		t.Run(tc.weapon, func(t *testing.T) {
			s, g := designBuildServer(t)
			player := g.Players[0]
			require.NotContains(t, append(append([]string{}, player.Available.Beams...), player.Available.Missiles...), tc.weapon)

			// The standalone editor endpoint stays research-agnostic.
			w := postCustomShipDesign(t, s, armedCustomShipPayload("Raider", tc.slot, tc.weapon))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			err := g.Turn(1, model.Commands{Build: []*model.BuildCommand{{PlanetID: 1, Building: "Raider"}}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), `cannot build "Raider"`)
			assert.Contains(t, err.Error(), tc.want)

			err = g.Turn(1, model.Commands{Build: []*model.BuildCommand{{PlanetID: 1, Building: "Flunker I", BuildQueue: []string{"Raider"}}}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), `cannot queue "Raider"`)
			assert.Contains(t, err.Error(), tc.want)

			for _, fleet := range g.Fleets {
				for _, ship := range fleet.Ships {
					assert.NotEqual(t, "Raider", ship.DesignName)
				}
			}
		})
	}
}

func TestInGameShipDesignSaveRejectsUnresearchedWeapons(t *testing.T) {
	for _, tc := range []struct{ slot, weapon, want string }{
		{"missile", "Zeon Missile", `missile \"Zeon Missile\" has not been researched`},
		{"beam", "Phasors", `beam \"Phasors\" has not been researched`},
	} {
		t.Run(tc.weapon, func(t *testing.T) {
			s, g := designBuildServer(t)
			w := designBuildRequest(t, s, g, http.MethodPost, "/resources/shipdesigns", armedCustomShipPayload("Raider", tc.slot, tc.weapon))
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), tc.want)
			assert.Empty(t, g.Players[0].ShipDesigns)
		})
	}
}
