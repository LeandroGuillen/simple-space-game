package model

import "encoding/json"

type Effect struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Bonus       int64  `json:"bonus,omitempty"`
}

func (e *Effect) UnmarshalJSON(data []byte) error {
	type effectAlias Effect
	var explicit effectAlias
	if err := json.Unmarshal(data, &explicit); err != nil {
		return err
	}
	*e = Effect(explicit)
	if e.Name != "" {
		return nil
	}

	var dynamic map[string]int64
	if err := json.Unmarshal(data, &dynamic); err != nil {
		return nil
	}
	for name, bonus := range dynamic {
		e.Name = name
		e.Bonus = bonus
		break
	}
	return nil
}
