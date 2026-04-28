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
                farmers: 2,
                workers: 3,
                scientists: 3,
                marines: 0,
                armor: 0,
                current_project: { id: 1, name: 'Automated Factory', cost: 60, maintenance: 1 },
                build_queue: null,
                buildings: null,
                industry_stack: 0,
                build_bucket: 60,
                is_building: true,
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
        building_collection: {
          buildings: [{ id: 1, name: 'Automated Factory', cost: 60, maintenance: 1 }],
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
    expect(game.galaxy.stars[0]?.ownedPlanets).toBe(1)
    expect(game.galaxy.stars[0]?.planets[0]?.maxPopulation).toBe(16000)
    expect(game.galaxy.stars[0]?.planets[0]?.currentProject?.name).toBe('Automated Factory')
  })
})
