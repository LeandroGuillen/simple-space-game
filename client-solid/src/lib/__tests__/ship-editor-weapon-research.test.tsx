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
  beams: [
    { name: "Laser", minDamage: 1, maxDamage: 4, required_tech: "Physics" },
    { name: "Phasors", minDamage: 6, maxDamage: 24, required_tech: "Multi-Phased Physics" },
  ],
  missiles: [
    { name: "Nuclear Missile", damage: 8, ammo: 5, speed: 12, required_tech: "Chemistry" },
    { name: "Zeon Missile", damage: 30, ammo: 5, speed: 12, required_tech: "Molecular Manipulation" },
  ],
};

it("the standalone Ship Editor shows every beam and missile's required tech and lock hint, and still saves any of them", async () => {
  const requests: Record<string, unknown>[] = [];
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    if (init?.method === "POST") {
      const body = JSON.parse(String(init.body));
      requests.push(body);
      return new Response(JSON.stringify({ custom_ship_design: { ...body, built_in: false } }));
    }
    return new Response(JSON.stringify(url.endsWith("/shipparts") ? parts : { ship_designs: [] }));
  });
  vi.stubGlobal("fetch", fetchMock);
  const view = render(() => <ShipEditorPage />);
  await waitFor(() => expect(view.getByRole("button", { name: "Clear fit-out" }).hasAttribute("disabled")).toBe(false));
  fireEvent.click(view.getByRole("button", { name: "New" }));
  fireEvent.click(view.getByRole("button", { name: "+ Add weapon" }));
  const select = view.getByRole("combobox", { name: "Weapon" }) as HTMLSelectElement;

  expect(Array.from(select.options, (option) => option.value)).toEqual(["Laser", "Phasors", "Nuclear Missile", "Zeon Missile"]);
  expect(Array.from(select.options, (option) => option.textContent)).toEqual([
    "Laser — 🔒 Requires Physics · locked until researched",
    "Phasors — 🔒 Requires Multi-Phased Physics · locked until researched",
    "Nuclear Missile — 🔒 Requires Chemistry · locked until researched",
    "Zeon Missile — 🔒 Requires Molecular Manipulation · locked until researched",
  ]);

  expect(view.getByRole("note").textContent).toBe("🔒 Requires Physics · locked until researched");
  fireEvent.change(select, { target: { value: "Zeon Missile" } });
  expect(view.getByRole("note").textContent).toBe("🔒 Requires Molecular Manipulation · locked until researched");

  // Research-agnostic: an unresearched missile still saves, with no game or player context.
  fireEvent.click(view.getByRole("button", { name: "Save design" }));
  await waitFor(() => expect(requests.at(-1)?.missile).toBe("Zeon Missile"));
  const post = fetchMock.mock.calls.find(([, init]) => init?.method === "POST")!;
  const headers = new Headers(post[1]?.headers);
  expect(headers.has("Game-Id")).toBe(false);
  expect(headers.has("Player-Id")).toBe(false);
});
