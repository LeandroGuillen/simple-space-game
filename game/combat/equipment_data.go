package combat

var HullSizes = []HullSize{
	{"Frigate", 25, 4, 10, 4, 1, 2, 1},
	{"Destroyer", 60, 8, 8, 10, 2, 5, 2},
	{"Cruiser", 120, 12, 6, 30, 5, 10, 6},
	{"Battleship", 250, 20, 5, 50, 7, 15, 8},
	{"Titan", 500, 30, 4, 80, 15, 30, 15},
	{"Doomstar", 1200, 50, 3, 150, 25, 50, 25},
}

var Drives = []Drive{
	{"Nuclear", 2, 0},
	{"Fusion", 3, 2},
	{"Ion", 4, 4},
	{"Anti-matter", 5, 6},
	{"Hyper", 6, 8},
	{"Interphased", 7, 10},
}

var Armors = []Armor{
	{"Titanium", 1, 5},
	{"Tritanium", 2, 10},
	{"Zortrium", 4, 15},
	{"Neutronium", 6, 25},
	{"Adamantium", 8, 30},
	{"Xentronium", 10, 40},
}

var Computers = []Computer{
	{"Electronic", 30},
	{"Optronic", 60},
	{"Positronic", 90},
	{"Cybertronic", 120},
	{"Molecultronic", 150},
}

var FuelCells = []FuelCell{
	{"Standard", 4},
	{"Deuterium", 6},
	{"Iridium", 9},
	{"Urridium", 12},
	{"Thorium", 100},
}

var Shields = []Shield{
	{"Class I", 0.3, 1, 1},
	{"Class III", 0.3, 3, 3},
	{"Class V", 0.3, 5, 5},
	{"Class VII", 0.3, 7, 7},
	{"Class X", 0.3, 10, 10},
}
