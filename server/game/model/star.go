package model

//go:generate stringer -type=StarColor
type StarColor byte

const (
	StarColorWhite StarColor = iota
	StarColorBlue
	StarColorYellow
	StarColorOrange
	StarColorRed
	StarColorMax
)

type Star struct {
	ID      uint      `json:"id"`
	Name    string    `json:"name,omitempty"`
	Planets []*Planet `json:"planets,omitempty"`
	Color   StarColor `json:"color"`
	X       int64     `json:"x"`
	Y       int64     `json:"y"`
}
