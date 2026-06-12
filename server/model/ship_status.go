package model

type ShipStatus struct {
	// hitpoints
	CurrentArmorHP     int64 `json:"current_armor_hp"`
	CurrentStructureHP int64 `json:"current_structure_hp"`
	MaxArmorHP         int64 `json:"max_armor_hp"`
	MaxStructureHP     int64 `json:"max_structure_hp"`

	// equipment hitpoints
	CurrentComputerHP int64 `json:"current_computer_hp"`
	CurrentDriveHP    int64 `json:"current_drive_hp"`
	CurrentShieldHP   int64 `json:"current_shield_hp"`
}
