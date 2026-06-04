# Terraforming Implementation Plan

## Context

The game already has scaffolded Terraforming and Gaia Transformation buildings in `buildings.json` and corresponding tech unlocks in `techs.buildings.json`, but no Go code implements the actual environment transformation. Design notes in `server/model/planet.go:178-202` define the intended transition graph. This plan implements the Terraforming building as a repeatable, consumed-on-use project that transforms a planet's environment one step toward a more habitable state, with cost scaling per use.

**Scope:** Terraforming only (up to Terran). Gaia Transformation is excluded.

## Transition Graph

```
Toxic    -> Barren              (requires Weather Controller on planet)
Radiated -> Barren              (requires Planetary Radiation Shield on planet)
Barren   -> Desert OR Tundra    (random 50/50)
Desert   -> Arid
Tundra   -> Swamp
Arid     -> Terran
Swamp    -> Terran
Ocean    -> Terran
```

## Cost

`250 + (planet.TerraformingCount * 250)` PP. Count increments after each completion.

---

## Changes

### 1. `server/model/planet.go` (line 46)

Add field to `Planet` struct after `Autobuild`:

```go
TerraformingCount int64 `json:"terraforming_count,omitempty"`
```

### 2. New file: `server/game/terraforming.go`

```go
package game

import (
    "math/rand"
    . "ssg/model"
)

const terraformingProject = "Terraforming"
const terraformingCostStep = int64(250)

var terraformTransitions = map[PlanetEnvironment][]PlanetEnvironment{
    PlanetEnvironmentToxic:    {PlanetEnvironmentBarren},
    PlanetEnvironmentRadiated: {PlanetEnvironmentBarren},
    PlanetEnvironmentBarren:   {PlanetEnvironmentDesert, PlanetEnvironmentTundra},
    PlanetEnvironmentDesert:   {PlanetEnvironmentArid},
    PlanetEnvironmentTundra:   {PlanetEnvironmentSwamp},
    PlanetEnvironmentArid:     {PlanetEnvironmentTerran},
    PlanetEnvironmentSwamp:    {PlanetEnvironmentTerran},
    PlanetEnvironmentOcean:    {PlanetEnvironmentTerran},
}

var terraformPrerequisites = map[PlanetEnvironment]string{
    PlanetEnvironmentToxic:    "Weather Controller",
    PlanetEnvironmentRadiated: "Planetary Radiation Shield",
}

func isTerraformingProject(name string) bool {
    return name == terraformingProject
}

func canTerraformPlanet(p *Planet, game *Game) bool {
    if p == nil || p.Owner == 0 {
        return false
    }
    targets, ok := terraformTransitions[p.Environment]
    if !ok || len(targets) == 0 {
        return false
    }
    if prereq, hasPrereq := terraformPrerequisites[p.Environment]; hasPrereq {
        if !hasBuilding(p, prereq) {
            return false
        }
    }
    return true
}

func terraformPlanet(p *Planet) {
    targets, ok := terraformTransitions[p.Environment]
    if !ok || len(targets) == 0 {
        return
    }
    p.Environment = targets[rand.Intn(len(targets))]
    p.TerraformingCount++
}

func effectiveTerraformingCost(p *Planet) int64 {
    return terraformingCostStep + p.TerraformingCount*terraformingCostStep
}
```

### 3. `server/game/turn.go` - 7 edits

**a. Build completion switch (after line 185, before `} else if completedProject == "Colony Ship"`):**

```go
} else if completedProject == terraformingProject {
    terraformPlanet(p)
```

**b. `isAutobuildProject()` (line 293-303) - add to exclusion list:**

Add `terraformingProject` to the switch case list.

**c. `Build()` (line 365-370) - add terraforming guard after artificial planet guard:**

```go
if isTerraformingProject(buildingName) && p.IsBuilding && p.CurrentProject == buildingName {
    return true
}
if isTerraformingProject(buildingName) && !canTerraformPlanet(p, game) {
    return false
}
```

**d. `canBuild()` (line 452-454) - add terraforming guard after artificial planet guard:**

```go
if isTerraformingProject(buildingName) {
    return canTerraformPlanet(p, game)
}
```

**e. `effectiveBuildingCost()` (line 492) - add terraforming cost override at top of function:**

```go
if isTerraformingProject(building.Name) {
    return effectiveTerraformingCost(p)
}
```

**f. `buildingApplicableToPlanet()` (line 572-576) - add terraforming case:**

```go
case terraformingProject:
    return canTerraformPlanet(p, game)
```

**g. `isRepeatableProject()` (line 608-610) - add terraforming:**

```go
func isRepeatableProject(buildingName string) bool {
    return buildingName == "Colony Base" || buildingName == "Colony Ship" || buildingName == "Freighters" || buildingName == artificialPlanetProject || isTerraformingProject(buildingName)
}
```

### 4. `server/game/resources/buildings.json` (line 353-358)

Update Terraforming description:

```json
"description": "Transforms the planet's environment one step toward a more habitable state. Cost increases by 250 PP with each use."
```

### 5. `client-solid/src/lib/types/api.ts` (after line 217)

Add to `ApiPlanet`:

```typescript
terraforming_count?: number;
```

### 6. `client-solid/src/lib/types/domain.ts` (after line 116)

Add to `PlanetViewModel`:

```typescript
terraformingCount: number;
```

### 7. `client-solid/src/lib/mappers.ts` (after line 418)

Add to `mapPlanet` return:

```typescript
terraformingCount: planet.terraforming_count ?? 0,
```

### 8. `client-solid/src/lib/construction.ts`

**a. Add to `AUTOBUILD_EXCLUDED_PROJECTS` (line 3-14):**

```typescript
'Terraforming',
```

**b. Extend `isBuildingApplicableToPlanet()` (line 36-45):**

Add terraforming check. The function signature needs `environment` and `buildings` added to the planet pick type:

```typescript
export function isBuildingApplicableToPlanet(
  project: Pick<BuildingViewModel, 'name'>,
  planet: Pick<PlanetViewModel, 'gravity' | 'environment' | 'buildings'>,
  raceTraitIds: string[] | undefined,
): boolean {
  if (project.name === GRAVITY_GENERATOR_BUILDING_NAME) {
    return planet.gravity !== preferredGravityForRace(raceTraitIds);
  }
  if (project.name === 'Terraforming') {
    return canTerraformPlanet(planet);
  }
  return true;
}
```

Add helper:

```typescript
const TERRAFORM_TRANSITIONS: Record<number, number[]> = {
  3: [5],   // Toxic -> Barren
  4: [5],   // Radiated -> Barren
  5: [6, 8], // Barren -> Desert or Tundra
  6: [7],   // Desert -> Arid
  8: [9],   // Tundra -> Swamp
  7: [11],  // Arid -> Terran
  9: [11],  // Swamp -> Terran
  10: [11], // Ocean -> Terran
};

const TERRAFORM_PREREQUISITES: Record<number, string> = {
  3: 'Weather Controller',
  4: 'Planetary Radiation Shield',
};

export function canTerraformPlanet(
  planet: Pick<PlanetViewModel, 'environment' | 'buildings'>,
): boolean {
  const targets = TERRAFORM_TRANSITIONS[planet.environment];
  if (!targets || targets.length === 0) return false;
  const prereq = TERRAFORM_PREREQUISITES[planet.environment];
  if (prereq && !planet.buildings.some((b) => b.name === prereq)) return false;
  return true;
}
```

### 9. `client-solid/src/components/ConstructionPanel.tsx`

**a. `isSpecialConstructionProject()` (line 86-96) - add:**

```typescript
projectName === "Terraforming" ||
```

**b. `isRepeatableProject` (line 543-548) - add:**

```typescript
projectName === "Terraforming" ||
```

**c. `sectionOneCommands` (line 370-404) - add Terraforming to commands list and availability gate:**

Add to the commands array:

```typescript
specialCommand("Terraforming", "Terraforming"),
```

Add availability filter in the `.map()` chain:

```typescript
if (command.unlockName === "Terraforming") {
  return {
    ...command,
    isAvailable: command.isAvailable && canTerraformPlanet(props.planet),
  };
}
```

Import `canTerraformPlanet` from `../lib/construction`.

**d. `effectiveProjectCost()` (line 444-460) - add terraforming cost override:**

```typescript
if (project.name === 'Terraforming') {
  return project.cost + (props.planet.terraformingCount ?? 0) * 250;
}
```

### 10. New file: `server/game/terraforming_test.go`

Tests covering:
- All valid transitions (Toxic->Barren, Radiated->Barren, Barren->Desert/Tundra, Desert->Arid, Tundra->Swamp, Arid->Terran, Swamp->Terran, Ocean->Terran)
- Prerequisite enforcement (Toxic without Weather Controller -> canTerraformPlanet returns false)
- Cost scaling (0 ops = 250, 1 op = 500, 2 ops = 750)
- TerraformingCount increments after each use
- Buildings preserved after transformation (Weather Controller stays after Toxic->Barren)
- Invalid environments (Gaia, Gas Giant, Terran, etc.) -> canTerraformPlanet returns false
- Build integration: terraforming project starts building, completes, and transforms planet

Test helper follows the `newArtificialPlanetGame` pattern from `artificial_planet_test.go`.

---

## Verification

```bash
cd server && go test ./...
cd server && go fmt ./...
cd client-solid && npm run build
```
