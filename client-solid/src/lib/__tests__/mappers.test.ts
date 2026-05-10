import { mapGame } from '../mappers'
import type { ApiGame } from '../types/api'

describe('mapGame', () => {
  it('maps game payloads into the UI view model', () => {
    const apiGame: ApiGame = {
      id: 'game-1',
      current_turn: 3500,
      galaxy: {
        stars: [
          {
            id: 7,
            name: 'Atria',
            color: 4,
            x: 100,
            y: 200,
            planets: [
              {
                id: 3,
                name: 'Atria I',
                owner: 1,
                parent_star: 7,
                mineral_class: 3,
                size: 4,
                gravity: 2,
                environment: 11,
                population: 8000,
                max_population: 18000,
                last_turn_pop_growth: 0,
                food: { farmers: 2, produced_per_farmer: 2, produced_per_building: 0 },
                industry: { workers: 3, produced_per_worker: 3, produced_per_building: 0 },
                science: { scientists: 3, produced_per_scientist: 3, produced_per_building: 0 },
                morale: {
                  bonus: 0,
                  multiplier: 1,
                  sources: [
                    { name: 'Holo Simulator', type: 'building', bonus: 0.2 },
                    { name: 'Penalty', type: 'building', bonus: -0.2 },
                  ],
                },
                pollution: 2,
                marines: 0,
                armor: 0,
                current_project: 'Automated Factory',
                build_queue: null,
                buildings: ['Capitol', 'Star Base'],
                industry_stack: 0,
                build_bucket: 60,
                is_building: true,
                autobuild: true,
              },
            ],
          },
        ],
      },
      players: [
        {
          id: 1,
          name: 'Ada',
          color: '#FFFFFF',
          race: { id: 0, name: 'Human' },
          treasury: 24,
          tax_rate: 0,
          research_points: 5,
          researched_techs: ['Starting Colony Infrastructure', 'Automated Factory'],
          available: {
            buildings: ['Colony Base', 'Automated Factory'],
            drives: ['Nuclear'],
          },
          fleets: [1],
          freighters: 1,
          freighters_used: 0,
          ship_designs: [],
        },
      ],
      resources: {
        ship_parts: {
          hull_sizes: [],
          drives: [],
          armors: [],
          computers: [],
          fuel_cells: [],
          shields: [],
        },
        ship_system_collection: {
          ship_systems: [
            {
              name: 'Reinforced Hull',
              description: 'Significantly increases ship structural strength.',
            },
          ],
        },
        building_collection: {
          buildings: [
            {
              name: 'Automated Factory',
              description: 'Robotic assembly lines.',
              icon: '🏭',
              cost: 60,
              maintenance: 1,
              grants: { modifiers: [{ name: 'industry_flat', value: 5 }] },
            },
            {
              name: 'Capitol',
              description: 'Central government building.',
              icon: '🏛️',
              cost: 0,
              maintenance: 0,
            },
            {
              name: 'Star Base',
              description: 'Orbital command station.',
              icon: '🛰️',
              cost: 200,
              maintenance: 2,
            },
          ],
        },
        ship_design_collection: {
          ship_designs: [],
        },
      },
    }

    const game = mapGame(apiGame)

    expect(game.id).toBe('game-1')
    expect(game.currentTurn).toBe(3500)
    expect(game.players[0]?.raceName).toBe('Human')
    expect(game.players[0]?.researchedTechs).toEqual(['Starting Colony Infrastructure', 'Automated Factory'])
    expect(game.players[0]?.available.buildings).toEqual(['Colony Base', 'Automated Factory'])
    expect(game.players[0]?.available.drives).toEqual(['Nuclear'])
    expect(game.players[0]?.availableBuildings).toEqual(['Colony Base', 'Automated Factory'])
    expect(game.resources.shipSystemCollection.shipSystems[0]?.name).toBe('Reinforced Hull')
    expect(game.galaxy.stars[0]?.ownedPlanets).toBe(1)
    expect(game.galaxy.stars[0]?.planets[0]?.maxPopulation).toBe(18000)
    expect(game.galaxy.stars[0]?.planets[0]?.pollution).toBe(2)
    expect(game.galaxy.stars[0]?.planets[0]?.currentProject?.name).toBe('Automated Factory')
    expect(game.galaxy.stars[0]?.planets[0]?.currentProject?.id).toBe(1)
    expect(game.galaxy.stars[0]?.planets[0]?.currentProject?.icon).toBe('🏭')
    expect(game.galaxy.stars[0]?.planets[0]?.autobuild).toBe(true)
    expect(game.galaxy.stars[0]?.planets[0]?.currentProject?.modifiers).toEqual([
      { name: 'industry_flat', description: '', bonus: 5 },
    ])
    expect(game.galaxy.stars[0]?.planets[0]?.buildings.map((building) => building.name)).toEqual([
      'Capitol',
      'Star Base',
    ])
    expect(game.galaxy.stars[0]?.planets[0]?.income.maintenance).toBe(2)
    expect(game.galaxy.stars[0]?.planets[0]?.income.total).toBe(6)
    expect(game.galaxy.stars[0]?.planets[0]?.morale).toEqual({
      bonus: 0,
      multiplier: 1,
      sources: [
        { name: 'Holo Simulator', type: 'building', bonus: 0.2 },
        { name: 'Penalty', type: 'building', bonus: -0.2 },
      ],
    })
  })
})
