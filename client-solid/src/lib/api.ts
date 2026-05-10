import type {
  ApiGame,
  ApiGameListItem,
  ApiGameUpdateMessage,
  ApiPlayerColorOptions,
  ApiShipParts,
  ApiStatusResponse,
  CustomRaceDesign,
  NewGameOptions,
} from './types/api';
import type { ShipDesignForm } from './types/domain';
import type { TurnCommands } from './types/commands';
import { normalizeServerUrl } from './utils';

type JsonHeaders = Record<string, string>;

async function readJson<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const body = (await response.json()) as {
        message?: string;
        error?: string;
      };
      message = body.message || body.error || message;
    } catch {
      // ignore malformed JSON error bodies
    }
    throw new Error(message);
  }
  return (await response.json()) as T;
}

export class ApiClient {
  readonly baseUrl: string;
  readonly playerId: number;

  constructor(baseUrl: string, playerId = 1) {
    this.baseUrl = normalizeServerUrl(baseUrl);
    this.playerId = playerId;
  }

  private headers(gameId?: string): HeadersInit {
    const headers: JsonHeaders = {
      'content-type': 'application/json',
    };
    if (gameId) {
      headers['Game-Id'] = gameId;
      headers['Player-Id'] = String(this.playerId);
    }
    return headers;
  }

  async status() {
    const response = await fetch(`${this.baseUrl}/status`);
    return readJson<ApiStatusResponse>(response);
  }

  async getGames() {
    const response = await fetch(`${this.baseUrl}/games`);
    const body = await readJson<{ game_list: ApiGameListItem[] }>(response);
    return body.game_list;
  }

  async getPlayerColorOptions() {
    const response = await fetch(`${this.baseUrl}/games/player-colors`);
    return readJson<ApiPlayerColorOptions>(response);
  }

  async getCustomRaces() {
    const response = await fetch(`${this.baseUrl}/races/custom`);
    const body = await readJson<{ custom_races: CustomRaceDesign[] }>(response);
    return body.custom_races;
  }

  async saveCustomRace(design: CustomRaceDesign) {
    const response = await fetch(`${this.baseUrl}/races/custom`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify({
        name: design.name,
        trait_ids: design.trait_ids,
      }),
    });
    const body = await readJson<{ custom_race: CustomRaceDesign }>(response);
    return body.custom_race;
  }

  async deleteCustomRace(name: string) {
    const response = await fetch(`${this.baseUrl}/races/custom/${encodeURIComponent(name)}`, {
      method: 'DELETE',
      headers: this.headers(),
    });
    return readJson<{ ok: boolean }>(response);
  }

  async createGame(options: NewGameOptions) {
    const response = await fetch(`${this.baseUrl}/games`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify(options),
    });
    const body = await readJson<{ ok: boolean; gameID: string }>(response);
    return body.gameID;
  }

  async getGame(gameId: string) {
    const response = await fetch(`${this.baseUrl}/games/${gameId}`, {
      headers: this.headers(gameId),
    });
    return readJson<ApiGame>(response);
  }

  async getShipParts(gameId: string) {
    const response = await fetch(`${this.baseUrl}/resources/shipparts`, {
      headers: this.headers(gameId),
    });
    return readJson<ApiShipParts>(response);
  }

  async getShipDesigns(gameId: string) {
    const response = await fetch(`${this.baseUrl}/resources/shipdesigns`, {
      headers: this.headers(gameId),
    });
    return readJson<unknown[]>(response);
  }

  async endTurn(gameId: string, commands: TurnCommands) {
    const response = await fetch(`${this.baseUrl}/turn`, {
      method: 'POST',
      headers: this.headers(gameId),
      body: JSON.stringify({
        game_id: gameId,
        commands,
      }),
    });
    return readJson<{ ok: boolean }>(response);
  }

  async colonize(gameId: string, planetId: number) {
    const response = await fetch(`${this.baseUrl}/action/${this.playerId}/colonize/${planetId}`, {
      method: 'POST',
      headers: this.headers(gameId),
      body: JSON.stringify({
        game_id: gameId,
      }),
    });
    return readJson<{ game: ApiGame }>(response);
  }

  async exploreStar(gameId: string, starId: number) {
    const response = await fetch(`${this.baseUrl}/action/${this.playerId}/explore/${starId}`, {
      method: 'POST',
      headers: this.headers(gameId),
      body: JSON.stringify({
        game_id: gameId,
      }),
    });
    return readJson<{ game: ApiGame }>(response);
  }

  async debugResearchAllTechs(gameId: string) {
    const response = await fetch(`${this.baseUrl}/debug/research-all`, {
      method: 'POST',
      headers: this.headers(gameId),
      body: JSON.stringify({
        game_id: gameId,
      }),
    });
    return readJson<{ game: ApiGame }>(response);
  }

  async saveShipDesign(gameId: string, form: ShipDesignForm, mode: 'create' | 'update') {
    const response = await fetch(`${this.baseUrl}/resources/shipdesigns`, {
      method: mode === 'create' ? 'POST' : 'PUT',
      headers: this.headers(gameId),
      body: JSON.stringify({
        name: form.name,
        drive: form.drive,
        computer: form.computer,
        shield: form.shield,
        armor: form.armor,
        hull_size: form.hullSize,
        fuel_cell: form.fuelCell,
      }),
    });
    return readJson<{ ok: boolean }>(response);
  }
}

export function parseGameUpdate(payload: string) {
  return JSON.parse(payload) as ApiGameUpdateMessage;
}
