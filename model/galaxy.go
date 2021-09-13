package model

import "fmt"

type Galaxy struct {
	Stars []*Star `json:"stars"`
}

func (g *Galaxy) String() string {
	return fmt.Sprintf("a galaxy with %d stars", len(g.Stars))
}
