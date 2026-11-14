import { isLobbyPath } from "./utils";

export const APP_NAME = "SSG";

const screenTitles: { prefix: string; title: string }[] = [
  { prefix: "/battle/", title: "Battle" },
  { prefix: "/stars/", title: "Planet" },
  { prefix: "/mutation", title: "Mutation" },
  { prefix: "/technology", title: "Technology" },
  { prefix: "/colonies", title: "Colonies" },
  { prefix: "/fleets", title: "Fleets" },
  { prefix: "/diplomacy", title: "Diplomacy" },
  { prefix: "/galaxy", title: "Galaxy" },
  { prefix: "/ship-editor", title: "Ship Editor" },
  { prefix: "/race-editor", title: "Race Editor" },
];

export function screenTitleForPath(pathname: string): string {
  if (pathname === "/" || pathname.startsWith("/setup")) {
    return "Setup";
  }

  for (const entry of screenTitles) {
    if (pathname.startsWith(entry.prefix)) {
      return entry.title;
    }
  }

  return "SSG";
}

export function gameLabelForDocument(
  game: { id: string; players: { id?: number; name: string }[] },
  playerId: number,
): string {
  const player =
    game.players.find((entry) => entry.id === playerId) ?? game.players[0];
  return player?.name?.trim() || game.id;
}

export function shouldUseGameTurnTitle(pathname: string, hasGame: boolean): boolean {
  if (!hasGame || isLobbyPath(pathname)) {
    return false;
  }
  return true;
}

export function formatDocumentTitle(options: {
  pathname: string;
  hasGame: boolean;
  game?: {
    id: string;
    currentTurn: number;
    players: { id?: number; name: string }[];
  } | null;
  playerId?: number;
}): string {
  const game = options.game;
  if (
    game &&
    shouldUseGameTurnTitle(options.pathname, options.hasGame) &&
    typeof options.playerId === "number"
  ) {
    const label = gameLabelForDocument(game, options.playerId);
    return `${APP_NAME} - ${label} - Turn ${game.currentTurn}`;
  }

  const screen = screenTitleForPath(options.pathname);
  if (screen === APP_NAME) {
    return APP_NAME;
  }
  return `${APP_NAME} - ${screen}`;
}
