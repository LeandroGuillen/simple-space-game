package model

import "fmt"

type Galaxy struct {
	Stars  []*Star `json:"stars"`
	Width  int64   `json:"width,omitempty"`
	Height int64   `json:"height,omitempty"`
}

func (g *Galaxy) String() string {
	return fmt.Sprintf("a galaxy with %d stars", len(g.Stars))
}
