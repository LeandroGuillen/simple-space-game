package combat

type ShipStatus struct {
	// hitpoints
	CurrentArmorHP     int64
	CurrentStructureHP int64
	MaxArmorHP         int64
	MaxStructureHP     int64

	// equipment hitpoints
	CurrentComputerHP int64
	CurrentDriveHP    int64
	CurrentShieldHP   int64
}
