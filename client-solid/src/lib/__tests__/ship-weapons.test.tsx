import { cleanup, fireEvent, render, waitFor } from "@solidjs/testing-library";
import { afterEach, expect, it, vi } from "vitest";
import { ShipEditorPage } from "../../routes/ShipEditorPage";
import { ApiClient } from "../api";

vi.mock("@solidjs/router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("../../state/app-context", () => ({ useAppContext: () => ({ state: { session: { serverUrl: "http://example.test", playerId: 1 } } }) }));
afterEach(() => { cleanup(); vi.unstubAllGlobals(); window.sessionStorage.clear(); });
const parts = { drives: [{name:"Nuclear", interstellar_speed:1, combat_speed:1}], armors:[{name:"Titanium", multiplier:1, ground_troop_strength:1}], computers:[{name:"None", beam_attack_bonus:0}], shields:[{name:"None", recharge_rate:0, reduce_damage:0, absortion_factor:0}], hull_sizes:[{name:"Frigate", space:10}], fuel_cells:[{name:"Standard",autonomy:4}], beams:[{name:"Laser",minDamage:1,maxDamage:4}] };
it("Add weapon equips a beam and Save persists it; utility saves are explicit", async () => {
 const requests: Record<string, unknown>[] = [];
 vi.stubGlobal("fetch", vi.fn(async (url: string, init?: RequestInit) => {
   if (init?.method === "POST") { const body = JSON.parse(String(init.body)); requests.push(body); return new Response(JSON.stringify({custom_ship_design:{...body,built_in:false}})); }
   return new Response(JSON.stringify(url.endsWith("/shipparts") ? parts : url.endsWith("/custom") ? {custom_ship_designs:[]} : {ship_designs:[]}));
 }));
 const view = render(() => <ShipEditorPage />);
 await waitFor(() => expect(view.getByRole("button", {name:"Clear fit-out"}).hasAttribute("disabled")).toBe(false));
 fireEvent.click(view.getByRole("button", {name:"New"}));
 expect(view.queryByRole("button", {name:"+ Add weapon"})).not.toBeNull();
 fireEvent.click(view.getByRole("button", {name:"+ Add weapon"}));
 expect((view.getByRole("combobox", {name:"Weapon"}) as HTMLSelectElement).value).toBe("Laser");
 fireEvent.click(view.getByRole("button", {name:"Remove weapon"}));
 expect(view.getByText(/Weaponless utility design/)).toBeTruthy();
 fireEvent.click(view.getByRole("button", {name:"+ Add weapon"}));
 fireEvent.click(view.getByRole("button", {name:"Save design"}));
 await waitFor(() => expect(requests[0]?.beam).toBe("Laser"));
});
it.each(["create", "update"] as const)("in-game %s sends the selected beam", async (mode) => {
 const fetchMock = vi.fn(async (_url: string, _init?: RequestInit) => new Response(JSON.stringify({ok:true}))); vi.stubGlobal("fetch", fetchMock);
 const form = {name:"Armed", drive:"Nuclear", computer:"None", shield:"None", armor:"Titanium", hullSize:"Frigate",fuelCell:"Standard", beam:"Laser"};
 await new ApiClient("http://example.test").saveShipDesign("game", form, mode);
 expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)).beam).toBe("Laser");
});
