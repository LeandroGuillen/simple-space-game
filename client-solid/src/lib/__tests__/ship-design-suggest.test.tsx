import { cleanup, fireEvent, render, waitFor, within } from "@solidjs/testing-library";
import { afterEach, expect, it, vi } from "vitest";
import { ShipDesignModal } from "../../components/ShipDesignModal";
import { ShipEditorPage } from "../../routes/ShipEditorPage";
import { ApiClient } from "../api";
import { mapGame } from "../mappers";
import type { ApiShipParts } from "../types/api";
import type { GameViewModel, ShipDesignForm } from "../types/domain";

const context = vi.hoisted(() => ({
  state: {
    session: { serverUrl: "http://example.test", playerId: 2, selectedGameId: "game" },
    game: null as GameViewModel | null,
    isBusy: false,
  },
  saveShipDesign: vi.fn(),
  suggestShipDesign: vi.fn(),
}));
vi.mock("@solidjs/router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("../../state/app-context", () => ({ useAppContext: () => context }));

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  context.state.game = null;
  context.state.session.selectedGameId = "game";
  window.sessionStorage.clear();
});

const parts: ApiShipParts = {
  drives: [
    { name: "Nuclear", interstellar_speed: 1, combat_speed: 0 },
    { name: "Fusion", interstellar_speed: 2, combat_speed: 1 },
  ],
  armors: [
    { name: "Titanium", multiplier: 1, ground_troop_strength: 5 },
    { name: "Tritanium", multiplier: 2, ground_troop_strength: 10 },
  ],
  computers: [
    { name: "None", beam_attack_bonus: 0 },
    { name: "Electronic", beam_attack_bonus: 30 },
  ],
  shields: [
    { name: "None", recharge_rate: 0, reduce_damage: 0, absortion_factor: 0 },
    { name: "Class I", recharge_rate: 0, reduce_damage: 1, absortion_factor: 0 },
  ],
  hull_sizes: [{ name: "Frigate", space: 25, size_rating: 1, marines: 1, base_speed: 1, base_hp: 10, computer_hp: 1, drive_hp: 1, shield_hp: 1 }],
  fuel_cells: [
    { name: "Standard", autonomy: 2 },
    { name: "Deuterium", autonomy: 4 },
  ],
  beams: [{ name: "Laser", minDamage: 1, maxDamage: 4 }],
  missiles: [{ name: "Nuclear Missile", damage: 8, ammo: 5, speed: 12 }],
};

const apiSuggestion = {
  name: "Suggested Frigate 1",
  beam: "",
  missile: "Nuclear Missile",
  drive: "Fusion",
  computer: "Electronic",
  shield: "Class I",
  armor: "Tritanium",
  hull_size: "Frigate",
  fuel_cell: "Deuterium",
};

const suggestionForm: ShipDesignForm = {
  name: "Suggested Frigate 1",
  beam: "",
  missile: "Nuclear Missile",
  drive: "Fusion",
  computer: "Electronic",
  shield: "Class I",
  armor: "Tritanium",
  hullSize: "Frigate",
  fuelCell: "Deuterium",
};

function gameWithResearch() {
  return mapGame({
    id: "game", current_turn: 1, galaxy: { stars: [] },
    players: [{ id: 2, name: "Ada", color: "#ffffff", available: { beams: ["Laser"], missiles: ["Nuclear Missile"] } }],
    resources: { ship_parts: parts },
  });
}

function pressed(view: ReturnType<typeof within>, label: string) {
  const group = within(view.getByRole("group", { name: label }));
  return group.getAllByRole("button").filter((b) => b.getAttribute("aria-pressed") === "true").map((b) => b.textContent);
}

it("ApiClient asks the server for the session player's suggestion and maps it to a form", async () => {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify({ suggestion: apiSuggestion })));
  vi.stubGlobal("fetch", fetchMock);
  const form = await new ApiClient("http://example.test", 2).suggestShipDesign("game");
  expect(form).toEqual(suggestionForm);
  const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
  expect(url).toBe("http://example.test/resources/shipdesigns/suggest");
  expect(init.method).toBe("POST");
  expect(init.headers).toMatchObject({ "Game-Id": "game", "Player-Id": "2" });
});

it("the in-game designer fills the form with a suggestion without saving it", async () => {
  context.state.game = gameWithResearch();
  context.suggestShipDesign.mockResolvedValue(suggestionForm);
  render(() => <ShipDesignModal isOpen mode="create" onClose={() => {}} />);
  const view = within(document.body);

  fireEvent.click(view.getByRole("button", { name: "Suggest design" }));
  await waitFor(() => expect((view.getByLabelText("Design Name") as HTMLInputElement).value).toBe("Suggested Frigate 1"));
  expect(context.suggestShipDesign).toHaveBeenCalledTimes(1);
  expect(context.saveShipDesign).not.toHaveBeenCalled();
  expect(pressed(view, "Weapon (None = utility ship, cannot fight)")).toEqual(["Nuclear Missile"]);
  expect(pressed(view, "Drive")).toEqual(["Fusion"]);
  expect(pressed(view, "Armor")).toEqual(["Tritanium"]);
  expect(pressed(view, "Computer")).toEqual(["Electronic"]);
  expect(pressed(view, "Shield")).toEqual(["Class I"]);
  expect(pressed(view, "Fuel")).toEqual(["Deuterium"]);
  expect(view.getByRole("status").textContent).toContain("Review and edit");

  // The draft stays editable and is only saved on an explicit save.
  fireEvent.click(within(view.getByRole("group", { name: "Drive" })).getByRole("button", { name: "Nuclear" }));
  fireEvent.click(view.getByRole("button", { name: "Save Design" }));
  await waitFor(() => expect(context.saveShipDesign).toHaveBeenCalledWith(
    { ...suggestionForm, drive: "Nuclear" }, "create",
  ));
});

it("the in-game designer keeps the current draft and shows the error when a suggestion fails", async () => {
  context.state.game = gameWithResearch();
  context.suggestShipDesign.mockRejectedValue(new Error("no suitable ship part: no researched interstellar drive is available"));
  render(() => <ShipDesignModal isOpen mode="create" onClose={() => {}} />);
  const view = within(document.body);
  fireEvent.input(view.getByLabelText("Design Name"), { target: { value: "My Draft" } });
  fireEvent.click(within(view.getByRole("group", { name: "Armor" })).getByRole("button", { name: "Tritanium" }));

  fireEvent.click(view.getByRole("button", { name: "Suggest design" }));
  await waitFor(() => expect(view.getByRole("alert").textContent).toContain("interstellar drive"));
  expect((view.getByLabelText("Design Name") as HTMLInputElement).value).toBe("My Draft");
  expect(pressed(view, "Armor")).toEqual(["Tritanium"]);
  expect(context.saveShipDesign).not.toHaveBeenCalled();
});

function stubEditorFetch(requests: { url: string; init?: RequestInit }[]) {
  vi.stubGlobal("fetch", vi.fn(async (url: string, init?: RequestInit) => {
    requests.push({ url, init });
    if (url.endsWith("/resources/shipdesigns/suggest")) {
      return new Response(JSON.stringify({ suggestion: apiSuggestion }));
    }
    return new Response(JSON.stringify(url.endsWith("/shipparts") ? parts : { ship_designs: [] }));
  }));
}

it("the standalone Ship Editor loads a suggestion for the session player as an unsaved draft", async () => {
  const requests: { url: string; init?: RequestInit }[] = [];
  stubEditorFetch(requests);
  const view = render(() => <ShipEditorPage />);
  await waitFor(() => expect(view.getByRole("button", { name: "Clear fit-out" }).hasAttribute("disabled")).toBe(false));

  fireEvent.click(view.getByRole("button", { name: "Suggest design" }));
  await waitFor(() => expect((view.getByPlaceholderText("Ship name") as HTMLInputElement).value).toBe("Suggested Frigate 1"));
  const suggest = requests.find((r) => r.url.endsWith("/resources/shipdesigns/suggest"));
  expect(suggest?.init?.headers).toMatchObject({ "Game-Id": "game", "Player-Id": "2" });
  expect(view.getByText("Unsaved draft")).toBeTruthy();
  expect(view.getByText(/Suggested design loaded/)).toBeTruthy();
  expect((view.getByRole("combobox", { name: "Weapon" }) as HTMLSelectElement).value).toBe("Nuclear Missile");
  expect(requests.some((r) => r.url.includes("/ship-designs/custom"))).toBe(false);
});

it("the standalone Ship Editor without a game shows an error and keeps the current draft", async () => {
  context.state.session.selectedGameId = "";
  const requests: { url: string; init?: RequestInit }[] = [];
  stubEditorFetch(requests);
  const view = render(() => <ShipEditorPage />);
  await waitFor(() => expect(view.getByRole("button", { name: "Clear fit-out" }).hasAttribute("disabled")).toBe(false));
  fireEvent.click(view.getByRole("button", { name: "New" }));
  const name = view.getByPlaceholderText("Ship name") as HTMLInputElement;
  fireEvent.input(name, { target: { value: "Keep Me" } });

  fireEvent.click(view.getByRole("button", { name: "Suggest design" }));
  await waitFor(() => expect(view.getByRole("alert").textContent).toContain("Open the editor from a game"));
  expect(name.value).toBe("Keep Me");
  expect(requests.some((r) => r.url.includes("/suggest"))).toBe(false);
});
