import { cleanup, fireEvent, render, waitFor, within } from "@solidjs/testing-library";
import { afterEach, expect, it, vi } from "vitest";
import { ShipDesignModal } from "../../components/ShipDesignModal";
import { ShipEditorPage } from "../../routes/ShipEditorPage";
import { mapGame } from "../mappers";
import type { ApiShipParts } from "../types/api";
import type { GameViewModel } from "../types/domain";

const context = vi.hoisted(() => ({
  state: {
    session: { serverUrl: "http://example.test", playerId: 1 },
    game: null as GameViewModel | null,
    isBusy: false,
  },
  saveShipDesign: vi.fn().mockResolvedValue(undefined),
}));
vi.mock("@solidjs/router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("../../state/app-context", () => ({ useAppContext: () => context }));

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  context.state.game = null;
  window.sessionStorage.clear();
});

const parts: ApiShipParts = {
  drives: [{ name: "Nuclear", interstellar_speed: 1, combat_speed: 1 }],
  armors: [{ name: "Titanium", multiplier: 1, ground_troop_strength: 1 }],
  computers: [{ name: "None", beam_attack_bonus: 0 }],
  shields: [{ name: "None", recharge_rate: 0, reduce_damage: 0, absortion_factor: 0 }],
  hull_sizes: [{ name: "Frigate", space: 10, size_rating: 1, marines: 1, base_speed: 1, base_hp: 10, computer_hp: 1, drive_hp: 1, shield_hp: 1 }],
  fuel_cells: [{ name: "Standard", autonomy: 4 }],
  beams: [
    { name: "Unimplemented", minDamage: 0, maxDamage: 0 },
    { name: "Invalid", minDamage: -2, maxDamage: -1 },
    { name: "Laser", minDamage: 1, maxDamage: 4 },
    { name: "Fusion Beam", minDamage: 2, maxDamage: 8 },
    { name: "Phasors", minDamage: 5, maxDamage: 20 },
  ],
};

it("the Ship Editor offers effective beams and submits each selected weapon", async () => {
  const requests: Record<string, unknown>[] = [];
  vi.stubGlobal("fetch", vi.fn(async (url: string, init?: RequestInit) => {
    if (init?.method === "POST") {
      const body = JSON.parse(String(init.body));
      requests.push(body);
      return new Response(JSON.stringify({ custom_ship_design: { ...body, built_in: false } }));
    }
    return new Response(JSON.stringify(url.endsWith("/shipparts") ? parts : { ship_designs: [] }));
  }));
  const view = render(() => <ShipEditorPage />);
  await waitFor(() => expect(view.getByRole("button", { name: "Clear fit-out" }).hasAttribute("disabled")).toBe(false));
  fireEvent.click(view.getByRole("button", { name: "New" }));
  fireEvent.click(view.getByRole("button", { name: "+ Add weapon" }));
  const select = view.getByRole("combobox", { name: "Weapon" }) as HTMLSelectElement;
  expect(Array.from(select.options, (option) => option.value)).toEqual(["Laser", "Fusion Beam", "Phasors"]);
  expect(select.value).toBe("Laser");
  for (const beam of ["Laser", "Fusion Beam", "Phasors"]) {
    fireEvent.change(select, { target: { value: beam } });
    fireEvent.click(view.getByRole("button", { name: "Save design" }));
    await waitFor(() => expect(requests.at(-1)?.beam).toBe(beam));
    await waitFor(() => expect(view.getByRole("button", { name: "Save design" }).hasAttribute("disabled")).toBe(false));
  }
});

it.each([{ beams: [] }, { beams: [parts.beams![0]!] }])("the Ship Editor disables adding a weapon when none are effective (%j)", async ({ beams }) => {
  vi.stubGlobal("fetch", vi.fn(async (url: string) => new Response(JSON.stringify(
    url.endsWith("/shipparts") ? { ...parts, beams } : { ship_designs: [] },
  ))));
  const view = render(() => <ShipEditorPage />);
  await waitFor(() => expect(view.getByRole("button", { name: "Clear fit-out" }).hasAttribute("disabled")).toBe(false));
  fireEvent.click(view.getByRole("button", { name: "New" }));
  expect(view.getByRole("button", { name: "+ Add weapon" }).hasAttribute("disabled")).toBe(true);
});

it.each([true, false])("the modal intersects research with the effective weapon catalog (catalog present: %s)", async (hasCatalog) => {
  context.state.game = mapGame({
    id: "game", current_turn: 1, galaxy: { stars: [] },
    players: [{ id: 1, name: "Ada", color: "#ffffff", available: {
      beams: ["Laser", "Fusion Beam", "Unimplemented", "Invalid", "Missing"],
    } }],
    resources: { ship_parts: { ...parts, beams: hasCatalog ? parts.beams : undefined } },
  });
  render(() => <ShipDesignModal isOpen mode="create" onClose={() => {}} />);
  const view = within(document.body);
  const group = within(view.getByRole("group", { name: "Weapon (None = utility ship, cannot fight)" }));
  const names = hasCatalog ? ["None", "Laser", "Fusion Beam"] : ["None"];
  expect(group.getAllByRole("button").map((button) => button.textContent)).toEqual(names);
  fireEvent.input(view.getByLabelText("Design Name"), { target: { value: "Armed" } });
  for (const name of names) {
    fireEvent.click(group.getByRole("button", { name }));
    fireEvent.click(view.getByRole("button", { name: "Save Design" }));
    await waitFor(() => expect(context.saveShipDesign).toHaveBeenLastCalledWith(
      expect.objectContaining({ beam: name === "None" ? "" : name }), "create",
    ));
  }
});
