package model

type Effect struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Bonus       int64  `json:"bonus,omitempty"`
}
