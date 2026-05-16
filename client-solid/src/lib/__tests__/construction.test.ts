import { describe, expect, it } from 'vitest';

import { isAutobuildProject, toggleQueuedConstructionProject } from '../construction';
import type { BuildingViewModel } from '../types/domain';

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
});
