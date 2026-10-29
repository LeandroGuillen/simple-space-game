package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scenarioDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("data", "scenarios")
	if _, err := os.Stat(dir); err != nil {
		dir = filepath.Join("..", "data", "scenarios")
	}
	_, err := os.Stat(dir)
	require.NoError(t, err, "bundled scenarios directory")
	return dir
}

func loadHydratedScenario(t *testing.T, name string) *Game {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(scenarioDir(t), name))
	require.NoError(t, err, name)
	var g Game
	require.NoError(t, json.Unmarshal(data, &g), name)
	HydrateLoadedGame(&g, resources)
	return &g
}

func emptySubsFor(g *Game) map[uint]Commands {
	subs := make(map[uint]Commands, len(g.Players))
	for _, p := range g.Players {
		if p != nil {
			subs[p.ID] = Commands{}
		}
	}
	return subs
}

func injectFailBeforeFleets(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { advanceTurnTestHook = nil })
	advanceTurnTestHook = func(stage string) error {
		if stage == "before_fleet_movements" {
			return errors.New("injected advance failure")
		}
		return nil
	}
}

func assertSnapshotRestoreByteIdentical(t *testing.T, g *Game) {
	t.Helper()
	before := captureGameJSON(t, g)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)
	snap := snapshotGameState(g)
	restoreGameState(g, snap)
	after := captureGameJSON(t, g)
	if !bytes.Equal(before, after) {
		t.Fatalf("snapshot/restore changed JSON (%d -> %d bytes)", len(before), len(after))
	}
	require.Equal(t, beforeSubs, g.TurnSubmissions)
}

func assertFailedAdvanceByteIdentical(t *testing.T, g *Game) {
	t.Helper()
	before := captureGameJSON(t, g)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)
	injectFailBeforeFleets(t)
	err := g.AdvanceTurn(emptySubsFor(g))
	require.Error(t, err)
	after := captureGameJSON(t, g)
	if !bytes.Equal(before, after) {
		t.Fatalf("failed AdvanceTurn changed JSON (%d -> %d bytes); nil/empty not preserved", len(before), len(after))
	}
	require.Equal(t, beforeSubs, g.TurnSubmissions)
}

func TestFailedAdvancePreservesNilEmptyOnBundledScenarios(t *testing.T) {
	// On 9709a83 these fail by assertion: empty build_queue / fleets become null
	// (c6f5f918-t3516.json grew ~164897 -> 164909).
	for _, name := range []string{
		"c6f5f918-t3516.json",
		"599e0ecb-t3500.json",
		"battle-demo.json",
	} {
		t.Run(name, func(t *testing.T) {
			g := loadHydratedScenario(t, name)
			assertFailedAdvanceByteIdentical(t, g)
		})
	}
}

func TestSnapshotRestorePreservesNilEmptyOnBundledScenarios(t *testing.T) {
	for _, name := range []string{
		"c6f5f918-t3516.json",
		"599e0ecb-t3500.json",
		"battle-demo.json",
	} {
		t.Run(name, func(t *testing.T) {
			g := loadHydratedScenario(t, name)
			assertSnapshotRestoreByteIdentical(t, g)
		})
	}
}

func battleDemoAfterBattle(t *testing.T) *Game {
	t.Helper()
	g := loadHydratedScenario(t, "battle-demo.json")
	subs := emptySubsFor(g)
	require.NoError(t, g.AdvanceTurn(subs))
	require.NotEmpty(t, g.PendingBattles, "turn 1 should open a tactical battle")
	require.NoError(t, g.AdvanceTurn(subs))
	require.Empty(t, g.PendingBattles)
	require.NotEmpty(t, g.BattleReports, "turn 2 auto-resolve should record a battle report")
	// Loser keeps an empty non-nil fleets slice — the nil/empty bug surface.
	var sawEmptyFleets bool
	for _, p := range g.Players {
		if p != nil && p.Fleets != nil && len(p.Fleets) == 0 {
			sawEmptyFleets = true
			break
		}
	}
	require.True(t, sawEmptyFleets, "expected a player with empty non-nil fleets after battle")
	return g
}

func TestFailedAdvancePreservesNilEmptyOnBattleDemoAfterBattle(t *testing.T) {
	g := battleDemoAfterBattle(t)
	assertFailedAdvanceByteIdentical(t, g)
}

func TestSnapshotRestorePreservesNilEmptyOnBattleDemoAfterBattle(t *testing.T) {
	g := battleDemoAfterBattle(t)
	assertSnapshotRestoreByteIdentical(t, g)
}

func TestCloneSlicePreservesNilAndEmpty(t *testing.T) {
	var nilSlice []uint
	empty := []uint{}
	assert.Nil(t, cloneSlice(nilSlice))
	got := cloneSlice(empty)
	require.NotNil(t, got)
	assert.Empty(t, got)
	assert.Equal(t, []uint{1, 2}, cloneSlice([]uint{1, 2}))
}

func TestCopyGroundForcePreservesNilSlices(t *testing.T) {
	force := GroundForce{OwnerID: 1} // Marines/Armored nil
	cp := copyGroundForce(force)
	assert.Nil(t, cp.Marines)
	assert.Nil(t, cp.Armored)
	force.Marines = []GroundUnit{}
	force.Armored = []GroundUnit{}
	cp = copyGroundForce(force)
	require.NotNil(t, cp.Marines)
	require.NotNil(t, cp.Armored)
	assert.Empty(t, cp.Marines)
	assert.Empty(t, cp.Armored)
}
