import type {
  ApiGame,
  ApiGameListItem,
  ApiTurnResponse,
  ApiGameUpdateMessage,
  ApiGovernment,
  ApiPlayerColorOptions,
  ApiShipParts,
  ApiStatusResponse,
  CustomRaceDesign,
  NewGameOptions,
  RaceDesign,
  ShipDesignTemplateRow,
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

  async getRaces() {
    const response = await fetch(`${this.baseUrl}/races`);
    const body = await readJson<{ races: RaceDesign[] }>(response);
    return body.races;
  }

  async getCustomRaces() {
    const response = await fetch(`${this.baseUrl}/races/custom`);
    const body = await readJson<{ custom_races: CustomRaceDesign[] }>(response);
    return body.custom_races;
  }

  async saveCustomRace(
    design: CustomRaceDesign,
    opts?: { renameFrom?: string; saveAsNew?: boolean },
  ) {
    const body: Record<string, unknown> = {
      name: design.name,
      government: design.government,
      trait_ids: design.trait_ids,
    };
    if (opts?.renameFrom) {
      body.rename_from = opts.renameFrom;
    }
    if (opts?.saveAsNew) {
      body.save_as_new = true;
    }
    const response = await fetch(`${this.baseUrl}/races/custom`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify(body),
    });
    const data = await readJson<{ custom_race: CustomRaceDesign }>(response);
    return data.custom_race;
  }

  async getGovernments() {
    const response = await fetch(`${this.baseUrl}/resources/governments`);
    const body = await readJson<{ governments: ApiGovernment[] }>(response);
    return body.governments;
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

  async getShipParts(gameId?: string) {
    const response = await fetch(`${this.baseUrl}/resources/shipparts`, {
      headers: gameId ? this.headers(gameId) : this.headers(),
    });
    return readJson<ApiShipParts>(response);
  }

  async getShipDesignTemplates() {
    const response = await fetch(`${this.baseUrl}/ship-designs`);
    const body = await readJson<{ ship_designs: ShipDesignTemplateRow[] }>(response);
    return body.ship_designs;
  }

  async getCustomShipDesignTemplates() {
    const response = await fetch(`${this.baseUrl}/ship-designs/custom`);
    const body = await readJson<{ custom_ship_designs: ShipDesignTemplateRow[] }>(response);
    return body.custom_ship_designs;
  }

  async saveCustomShipDesign(
    design: ShipDesignForm,
    opts?: { renameFrom?: string; saveAsNew?: boolean },
  ) {
    const body: Record<string, unknown> = {
      name: design.name,
      drive: design.drive,
      computer: design.computer,
      shield: design.shield,
      armor: design.armor,
      hull_size: design.hullSize,
      fuel_cell: design.fuelCell,
    };
    if (opts?.renameFrom) {
      body.rename_from = opts.renameFrom;
    }
    if (opts?.saveAsNew) {
      body.save_as_new = true;
    }
    const response = await fetch(`${this.baseUrl}/ship-designs/custom`, {
      method: 'POST',
      headers: this.headers(),
      body: JSON.stringify(body),
    });
    const data = await readJson<{ custom_ship_design: ShipDesignTemplateRow }>(response);
    return data.custom_ship_design;
  }

  async deleteCustomShipDesign(name: string) {
    const response = await fetch(
      `${this.baseUrl}/ship-designs/custom/${encodeURIComponent(name)}`,
      {
        method: 'DELETE',
        headers: this.headers(),
      },
    );
    return readJson<{ ok: boolean }>(response);
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
    return readJson<ApiTurnResponse>(response);
  }

  async setPlanetAutobuild(gameId: string, planetId: number, enabled: boolean) {
    const response = await fetch(`${this.baseUrl}/planets/${planetId}/autobuild`, {
      method: 'PUT',
      headers: this.headers(gameId),
      body: JSON.stringify({ enabled }),
    });
    return readJson<ApiGame>(response);
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
