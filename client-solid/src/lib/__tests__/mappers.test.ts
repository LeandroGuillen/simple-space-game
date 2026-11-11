import { mapGame } from '../mappers'
import type { ApiGame } from '../types/api'

describe('mapGame', () => {
  it('maps game payloads into the UI view model', () => {
    const apiGame: ApiGame = {
      id: 'game-1',
      current_turn: 3500,
      status: 'finished',
      winner_id: 1,
      turns_played: 12,
      galaxy: {
        width: 990,
        height: 566,
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
      pending_artificial_planets: { 1: [3] },
      transfer_losses: [{ owner: 1, turn: 3500, transfer_id: 5, source_planet_id: 3, source_planet_name: 'Sol', destination_planet_id: 4, destination_planet_name: 'Vega', population_lost: 500 }],
      population_transfers: [
        {
          id: 9,
          owner: 1,
          source_planet_id: 3,
          destination_planet_id: 4,
          population_units: 1,
          freighters_reserved: 5,
          origin_star_id: 7,
          destination_star_id: 8,
          total_turns: 2,
          turns_remaining: 1,
        },
      ],
      players: [
        {
          id: 1,
          eliminated: true,
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
    expect(game.transferLosses).toEqual([{ owner: 1, turn: 3500, transferId: 5, sourcePlanetId: 3, sourcePlanetName: 'Sol', destinationPlanetId: 4, destinationPlanetName: 'Vega', populationLost: 500 }])
    expect(game.currentTurn).toBe(3500)
    expect(game.status).toBe('finished')
    expect(game.winnerId).toBe(1)
    expect(game.turnsPlayed).toBe(12)
    expect(game.players[0]?.eliminated).toBe(true)
    expect(game.galaxy.width).toBe(990)
    expect(game.galaxy.height).toBe(566)
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
    apiGame.galaxy.stars[0]!.planets![0]!.income = {
      taxes: 8, buildings: 0, trade_goods: 0, industry_tax: 3,
      maintenance: 2, total: 9, bonus: 0,
    }
    const taxed = mapGame(apiGame).galaxy.stars[0]!.planets[0]!.income
    expect(taxed.industryTax).toBe(3)
    expect(taxed.total).toBe(9)

    expect(game.galaxy.stars[0]?.planets[0]?.morale).toEqual({
      bonus: 0,
      multiplier: 1,
      sources: [
        { name: 'Holo Simulator', type: 'building', bonus: 0.2 },
        { name: 'Penalty', type: 'building', bonus: -0.2 },
      ],
    })
    expect(game.pendingArtificialPlanets).toEqual({ 1: [3] })
    expect(game.populationTransfers?.[0]).toEqual({
      id: 9,
      owner: 1,
      sourcePlanetId: 3,
      destinationPlanetId: 4,
      populationUnits: 1,
      freightersReserved: 5,
      originStarId: 7,
      destinationStarId: 8,
      totalTurns: 2,
      turnsRemaining: 1,
    })
  })

  it('defaults missing galaxy dimensions for old payloads', () => {
    const apiGame: ApiGame = {
      id: 'game-legacy',
      current_turn: 3500,
      galaxy: { stars: [] },
      players: [],
      fleets: [],
      resources: {
        ship_parts: {
          hull_sizes: [],
          drives: [],
          armors: [],
          computers: [],
          fuel_cells: [],
          shields: [],
        },
      },
    }

    const game = mapGame(apiGame)

    expect(game.galaxy.width).toBe(700)
    expect(game.galaxy.height).toBe(400)
  })
})

  it('tolerates rival players with private fields omitted', () => {
    const apiGame: ApiGame = {
      id: 'game-fog',
      current_turn: 3500,
      galaxy: { stars: [] },
      players: [
        {
          id: 1,
          name: 'Ada',
          color: '#FFFFFF',
          race: { id: 0, name: 'Human' },
          treasury: 24,
          tax_rate: 0,
          research_points: 5,
          researched_techs: ['Automated Factory'],
          available: { buildings: ['Colony Base'] },
          fleets: [1],
          freighters: 2,
          freighters_used: 1,
          ship_designs: [],
        },
        {
          id: 2,
          name: 'Rival',
          color: '#FF0000',
          race: { id: 0, name: 'Mrrshan' },
          // Fog-of-war: private empire fields intentionally omitted.
        },
      ],
      fleets: [
        {
          id: 1,
          owner: 1,
          star_id: 1,
          x: 0,
          y: 0,
          ships: [{ type: 'ship_design', name: 'Scout' }],
        },
        {
          id: 2,
          owner: 2,
          star_id: 1,
          x: 10,
          y: 10,
          // Enemy destination/orders omitted by the server.
          ships: [{ type: 'ship_design', name: 'Raider' }],
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
      },
    }

    const game = mapGame(apiGame)
    expect(game.players[1]?.name).toBe('Rival')
    expect(game.players[1]?.treasury).toBe(0)
    expect(game.players[1]?.researchPoints).toBe(0)
    expect(game.players[1]?.researchedTechs).toEqual([])
    expect(game.players[1]?.freighters).toBe(0)
    expect(game.players[1]?.availableBuildings).toEqual([])
    expect(game.fleets[1]?.destinationStarId).toBeNull()
    expect(game.fleets[1]?.turnsRemaining).toBe(0)
  })
