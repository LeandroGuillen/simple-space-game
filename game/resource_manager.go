package game

import (
	_ "embed"
	"encoding/json"
	"log"
)

type ResourceManager struct {
	equipment *PartListResponse
}

//go:embed resources/equipment.json
var equipmentFile []byte

func (r *ResourceManager) loadDataFromFile() {
	var response PartListResponse
	err := json.Unmarshal(equipmentFile, &response)
	if err != nil {
		log.Fatal(err)
	}
	r.equipment = &response
}

func (r *ResourceManager) GetShipPartsList() (PartListResponse, error) {
	if r.equipment == nil {
		r.loadDataFromFile()
	}
	var response PartListResponse = PartListResponse{
		HullSizes: r.equipment.HullSizes,
		Drives:    r.equipment.Drives,
		Armors:    r.equipment.Armors,
		Computers: r.equipment.Computers,
		FuelCells: r.equipment.FuelCells,
		Shields:   r.equipment.Shields,
	}

	return response, nil
}
