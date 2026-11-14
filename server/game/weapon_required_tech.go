package game

import (
	. "ssg/model"
)

// weaponRequiredTechs maps each beam and missile name to the tech-tree level
// whose research unlocks it (e.g. "Nuclear Missile" -> "Chemistry").
func weaponRequiredTechs(tree *TechTree) map[string]string {
	required := map[string]string{}
	if tree == nil {
		return required
	}
	techs := make(map[string]Tech, len(tree.AllTechs))
	for _, tech := range tree.AllTechs {
		techs[tech.Name] = tech
	}
	for _, subject := range tree.Subjects {
		for _, level := range subject.Levels {
			for _, techName := range level.AvailableTechs {
				tech, ok := techs[techName]
				if !ok {
					continue
				}
				unlocks := tech.Grants.Unlocks
				for _, weapon := range append(append([]string{}, unlocks.Beams...), unlocks.Missiles...) {
					if _, seen := required[weapon]; !seen {
						required[weapon] = level.Name
					}
				}
			}
		}
	}
	return required
}

// annotateWeaponRequiredTechs fills RequiredTech on the editor's beam and
// missile options. It is informational only: the standalone editor has no
// player, so it shows what must be researched without checking it.
func annotateWeaponRequiredTechs(parts *ShipPartsCollection, tree *TechTree) {
	if parts == nil {
		return
	}
	required := weaponRequiredTechs(tree)
	for i := range parts.BeamOptions {
		parts.BeamOptions[i].RequiredTech = required[parts.BeamOptions[i].Name]
	}
	for i := range parts.MissileOptions {
		parts.MissileOptions[i].RequiredTech = required[parts.MissileOptions[i].Name]
	}
}
