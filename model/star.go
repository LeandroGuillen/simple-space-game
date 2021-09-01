package model

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
	Id      int       `json:"id"`
	Name    string    `json:"name"`
	Planets []Planet  `json:"planets"`
	Color   StarColor `json:"color"`
	X       int64     `json:"x"`
	Y       int64     `json:"y"`
}
