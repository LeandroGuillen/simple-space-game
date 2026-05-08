package model

type Building struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Cost        int64  `json:"cost,omitempty"`
	Maintenance int64  `json:"maintenance,omitempty"`
	Unique      bool   `json:"unique,omitempty"`
	BuiltOn     string `json:"built_on,omitempty"`
	Grants      Grants `json:"grants,omitempty"`
}
