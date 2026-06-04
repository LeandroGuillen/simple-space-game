import { describe, expect, it } from 'vitest';

import {
  GRAVITY_GENERATOR_BUILDING_NAME,
  isAutobuildProject,
  isBuildingApplicableToPlanet,
  preferredGravityForRace,
  toggleQueuedConstructionProject,
} from '../construction';
import type { BuildingViewModel, PlanetViewModel } from '../types/domain';

function project(id: number, name: string): BuildingViewModel {
  return {
    id,
    name,
    description: '',
    icon: '',
    cost: 10,
    maintenance: 0,
    unique: false,
    upgrades: [],
    grants: {
      unlocks: {
        buildings: [],
        hullSizes: [],
        drives: [],
        armors: [],
        computers: [],
        fuelCells: [],
        shields: [],
        beams: [],
        shipSystems: [],
      },
      modifiers: [],
    },
    modifiers: [],
  };
}

describe('isAutobuildProject', () => {
  it('excludes colonization, ship, android, and spy projects', () => {
    expect(isAutobuildProject({ name: 'Automated Factory' })).toBe(true);
    expect(isAutobuildProject({ name: 'Research Lab' })).toBe(true);

    expect(isAutobuildProject({ name: 'Colony Base' })).toBe(false);
    expect(isAutobuildProject({ name: 'Colony Ship' })).toBe(false);
    expect(isAutobuildProject({ name: 'Artificial Planet Construction' })).toBe(false);
    expect(isAutobuildProject({ name: 'Outpost Ship' })).toBe(false);
    expect(isAutobuildProject({ name: 'Transport' })).toBe(false);
    expect(isAutobuildProject({ name: 'Freighters' })).toBe(false);
    expect(isAutobuildProject({ name: 'Spy' })).toBe(false);
    expect(isAutobuildProject({ name: 'Android Farmers' })).toBe(false);
    expect(isAutobuildProject({ name: 'Android Workers' })).toBe(false);
    expect(isAutobuildProject({ name: 'Android Scientists' })).toBe(false);
  });

  it('excludes future ship projects by name', () => {
    expect(isAutobuildProject({ name: 'Scout Ship' })).toBe(false);
  });
});

describe('preferredGravityForRace', () => {
  it('defaults to normal gravity', () => {
    expect(preferredGravityForRace(undefined)).toBe(2);
    expect(preferredGravityForRace([])).toBe(2);
    expect(preferredGravityForRace(['large_homeworld'])).toBe(2);
  });

  it('returns low gravity for low-G races', () => {
    expect(preferredGravityForRace(['low_g_world'])).toBe(1);
  });

  it('returns high gravity for high-G races', () => {
    expect(preferredGravityForRace(['high_g_world'])).toBe(3);
  });
});

describe('isBuildingApplicableToPlanet', () => {
  function planet(gravity: number): Pick<PlanetViewModel, 'gravity' | 'environment' | 'buildings'> {
    return { gravity, environment: 11, buildings: [] };
  }

  it('hides Gravity Generator when planet gravity matches the race', () => {
    expect(
      isBuildingApplicableToPlanet(
        { name: GRAVITY_GENERATOR_BUILDING_NAME },
        planet(2),
        [],
      ),
    ).toBe(false);
    expect(
      isBuildingApplicableToPlanet(
        { name: GRAVITY_GENERATOR_BUILDING_NAME },
        planet(1),
        ['low_g_world'],
      ),
    ).toBe(false);
    expect(
      isBuildingApplicableToPlanet(
        { name: GRAVITY_GENERATOR_BUILDING_NAME },
        planet(3),
        ['high_g_world'],
      ),
    ).toBe(false);
  });

  it('shows Gravity Generator when planet gravity differs from preferred', () => {
    expect(
      isBuildingApplicableToPlanet(
        { name: GRAVITY_GENERATOR_BUILDING_NAME },
        planet(1),
        [],
      ),
    ).toBe(true);
    expect(
      isBuildingApplicableToPlanet(
        { name: GRAVITY_GENERATOR_BUILDING_NAME },
        planet(3),
        [],
      ),
    ).toBe(true);
    expect(
      isBuildingApplicableToPlanet(
        { name: GRAVITY_GENERATOR_BUILDING_NAME },
        planet(2),
        ['low_g_world'],
      ),
    ).toBe(true);
  });

  it('keeps unrelated buildings applicable regardless of gravity', () => {
    expect(
      isBuildingApplicableToPlanet({ name: 'Automated Factory' }, planet(1), []),
    ).toBe(true);
  });

  function hostilePlanet(): Pick<
    PlanetViewModel,
    'gravity' | 'environment' | 'buildings'
  > {
    return { gravity: 2, environment: 3, buildings: [] };
  }

  const farmerBonusBuilding = {
    name: 'Soil Enrichment',
    grants: { modifiers: [{ name: 'food_per_farmer', value: 1 }] },
  };
  const flatFoodBuilding = {
    name: 'Hydroponic Farm',
    grants: { modifiers: [{ name: 'food_flat', value: 2 }] },
  };

  it('hides farmer-bonus-only buildings on hostile worlds without Biomorphic Fungi', () => {
    expect(
      isBuildingApplicableToPlanet(farmerBonusBuilding, hostilePlanet(), []),
    ).toBe(false);
  });

  it('shows farmer-bonus-only buildings on hostile worlds once Biomorphic Fungi is researched', () => {
    expect(
      isBuildingApplicableToPlanet(
        farmerBonusBuilding,
        hostilePlanet(),
        [],
        new Set(['Biomorphic Fungi']),
      ),
    ).toBe(true);
  });

  it('keeps farmer-bonus-only buildings applicable on non-hostile worlds', () => {
    expect(
      isBuildingApplicableToPlanet(farmerBonusBuilding, planet(2), []),
    ).toBe(true);
  });

  it('keeps flat-food buildings applicable on hostile worlds', () => {
    expect(
      isBuildingApplicableToPlanet(flatFoodBuilding, hostilePlanet(), []),
    ).toBe(true);
  });
});

describe('toggleQueuedConstructionProject', () => {
  it('adds a project that is not queued', () => {
    const factory = project(1, 'Automated Factory');

    expect(toggleQueuedConstructionProject([], factory)).toEqual([factory]);
  });

  it('removes a non-repeatable project that is already queued', () => {
    const factory = project(1, 'Automated Factory');
    const lab = project(2, 'Research Lab');

    expect(toggleQueuedConstructionProject([factory, lab], factory)).toEqual([
      lab,
    ]);
  });

  it('allows duplicate repeatable projects', () => {
    const colonyBase = project(3, 'Colony Base');

    expect(
      toggleQueuedConstructionProject([colonyBase], colonyBase, {
        isRepeatable: true,
      }),
    ).toEqual([colonyBase, colonyBase]);
  });

  it('replaces a lone Trade Goods entry when adding another project', () => {
    const tradeGoods = project(0, 'Trade Goods');
    const factory = project(1, 'Automated Factory');

    expect(
      toggleQueuedConstructionProject([tradeGoods], factory, {
        tradeGoodsId: 0,
      }),
    ).toEqual([factory]);
  });

  it('replaces a lone Housing entry when adding another project', () => {
    const housing = project(-1, 'Housing');
    const factory = project(1, 'Automated Factory');

    expect(
      toggleQueuedConstructionProject([housing], factory, {
        terminalProjectIds: [0, -1],
      }),
    ).toEqual([factory]);
  });
});
