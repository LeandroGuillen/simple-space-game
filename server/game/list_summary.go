package game

import "time"

// GameListPlayerSummary is a lock-safe snapshot of one player for the lobby list.
type GameListPlayerSummary struct {
	ID         uint
	Name       string
	RaceName   string
	Color      string
	NumStars   int
	Controller string
}

// GameListSummary is a lock-safe snapshot of lobby fields for one game.
type GameListSummary struct {
	ID          string
	NumStars    int
	CurrentTurn float64
	CreatedAt   time.Time
	LastTurnAt  *time.Time
	Debug       bool
	Players     []GameListPlayerSummary
}

// ListSummary returns lobby metadata under the read lock so concurrent turn
// processing cannot race the handler that builds GET /games.
func (g *Game) ListSummary() GameListSummary {
	g.ensureMu()
	g.mu.RLock()
	defer g.mu.RUnlock()

	players := make([]GameListPlayerSummary, 0, len(g.Players))
	for _, p := range g.Players {
		if p == nil {
			continue
		}
		ownedStars := 0
		for _, star := range g.Galaxy.Stars {
			if star == nil {
				continue
			}
			for _, planet := range star.Planets {
				if planet != nil && planet.Owner == p.ID {
					ownedStars++
					break
				}
			}
		}
		players = append(players, GameListPlayerSummary{
			ID:         p.ID,
			Name:       p.Name,
			RaceName:   p.Race.Name,
			Color:      p.Color,
			NumStars:   ownedStars,
			Controller: p.Controller,
		})
	}

	summary := GameListSummary{
		ID:          g.ID,
		NumStars:    len(g.Galaxy.Stars),
		CurrentTurn: g.CurrentTurn,
		CreatedAt:   g.CreatedAt,
		Debug:       g.Debug,
		Players:     players,
	}
	if g.LastTurnAt != nil {
		t := *g.LastTurnAt
		summary.LastTurnAt = &t
	}
	return summary
}
