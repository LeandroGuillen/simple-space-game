package game

// MarkExternalController lets an external agent take over a server AI slot.
// Legacy slots with no controller retain their existing behavior.
func (g *Game) MarkExternalController(playerID uint) {
	g.ensureMu()
	g.mu.Lock()
	defer g.mu.Unlock()
	if player, err := g.getPlayer(playerID); err == nil && player.Controller == "server_ai" {
		player.Controller = "external"
	}
}
