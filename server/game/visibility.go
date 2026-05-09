package game

import (
	"ssg/model"
)

func (g *Game) ExploreStar(playerID, starID uint) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}
	if _, err := g.getStar(starID); err != nil {
		return err
	}
	if hasExploredStar(player.ExploredStars, starID) {
		return nil
	}
	player.ExploredStars = append(player.ExploredStars, starID)
	return nil
}

func (g *Game) GetVisibleGameForPlayer(playerID uint) (*Game, error) {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return nil, err
	}

	g.HydrateCommandPoints()

	visible := *g
	visible.Galaxy = model.Galaxy{Stars: make([]*model.Star, 0, len(g.Galaxy.Stars))}
	for _, star := range g.Galaxy.Stars {
		starCopy := &model.Star{
			ID:    star.ID,
			Color: star.Color,
			X:     star.X,
			Y:     star.Y,
		}
		if hasExploredStar(player.ExploredStars, star.ID) {
			starCopy.Name = star.Name
			starCopy.Planets = make([]*model.Planet, 0, len(star.Planets))
			for _, planet := range star.Planets {
				planetCopy := *planet
				if planet.Owner != 0 && planet.Owner == playerID {
					hydratePlanetOutputs(g, &planetCopy)
					hydratePlanetPopulationCapacity(g, &planetCopy, player)
				}
				starCopy.Planets = append(starCopy.Planets, &planetCopy)
			}
		}
		visible.Galaxy.Stars = append(visible.Galaxy.Stars, starCopy)
	}

	return &visible, nil
}

func hasExploredStar(explored []uint, starID uint) bool {
	for _, id := range explored {
		if id == starID {
			return true
		}
	}
	return false
}
