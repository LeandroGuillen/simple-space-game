import { cleanup, fireEvent, render, waitFor } from "@solidjs/testing-library";
import { afterEach, expect, it, vi } from "vitest";
import { ShipEditorPage } from "../../routes/ShipEditorPage";

vi.mock("@solidjs/router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("../../state/app-context", () => ({
  useAppContext: () => ({ state: { session: { serverUrl: "http://example.test", playerId: 1 } } }),
}));
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  window.sessionStorage.clear();
});

const parts = {
  drives: [{ name: "Nuclear", interstellar_speed: 1, combat_speed: 1 }],
  armors: [{ name: "Titanium", multiplier: 1, ground_troop_strength: 1 }],
  computers: [{ name: "None", beam_attack_bonus: 0 }],
  shields: [{ name: "None", recharge_rate: 0, reduce_damage: 0, absortion_factor: 0 }],
  hull_sizes: [{ name: "Frigate", space: 10 }],
  fuel_cells: [{ name: "Standard", autonomy: 4 }],
  beams: [{ name: "Laser", minDamage: 1, maxDamage: 4 }],
  missiles: [{ name: "Nuclear Missile", damage: 8, ammo: 5, speed: 12 }],
};

it("the Ship Editor offers missiles beside beams and saves a missile rack", async () => {
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
  expect(Array.from(select.options, (option) => option.value)).toEqual(["Laser", "Nuclear Missile"]);
  expect(Array.from(select.querySelectorAll("optgroup"), (group) => group.label)).toEqual(["Beams", "Missiles"]);

  fireEvent.change(select, { target: { value: "Nuclear Missile" } });
  expect(view.getByText("8 (5 shots/battle)")).toBeTruthy();
  fireEvent.click(view.getByRole("button", { name: "Save design" }));
  await waitFor(() => expect(requests.at(-1)?.missile).toBe("Nuclear Missile"));
  expect(requests.at(-1)?.beam).toBe("");
});
