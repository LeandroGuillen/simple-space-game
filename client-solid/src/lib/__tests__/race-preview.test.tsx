import { cleanup, fireEvent, render } from "@solidjs/testing-library";
import { createSignal } from "solid-js";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RacePreview } from "../../components/RacePreview";
import { OptionRow } from "../../routes/SetupPage";
import { fallbackGovernments } from "../race-design";
import type { RaceDesign } from "../types/api";

afterEach(cleanup);
const human: RaceDesign = { name: "Human", built_in: true, government: "Democracy", trait_ids: [] };
const custom: RaceDesign = { name: "Custom", government: "Dictatorship", trait_ids: ["research_plus_2", "low_g_world", "creative"] };

describe("race preview", () => {
  it("shows government, grouped traits, descriptions, partial status, and custom pick cost", () => {
    const edit = vi.fn();
    const view = render(() => <RacePreview race={custom} governments={fallbackGovernments} selected onEdit={edit} />);
    expect(view.getByText("Dictatorship")).toBeTruthy();
    expect(view.getByText(/centralized authoritarian government/)).toBeTruthy();
    expect(view.getByText("Science")).toBeTruthy();
    expect(view.getByText("Special")).toBeTruthy();
    expect(view.getByText("+2 Research")).toBeTruthy();
    expect(view.getByText("Low-G World")).toBeTruthy();
    expect(view.getByText("Creative")).toBeTruthy();
    expect(view.getByText(/Scientists produce two extra research/)).toBeTruthy();
    expect(view.getByText("Partially implemented").getAttribute("title")).toContain("troop penalty");
    expect(view.getByText("Picks used: 9")).toBeTruthy();
    fireEvent.click(view.getByRole("button", { name: "Edit in Race Editor" }));
    expect(edit).toHaveBeenCalledWith(custom);
  });
  it("previews on focus and hover without changing selection, and selects on click", () => {
    const view = render(() => {
      const [selected, setSelected] = createSignal("human");
      const [preview, setPreview] = createSignal("human");
      return <>
        <OptionRow label="Race" options={[{ label: "Human", value: "human" }, { label: "Custom", value: "custom" }]} selected={selected()} onSelect={(value) => { setSelected(value); setPreview(value); }} onPreview={setPreview} />
        <RacePreview race={preview() === "human" ? human : custom} governments={fallbackGovernments} selected={selected() === preview()} onEdit={() => {}} />
      </>;
    });
    const humanButton = view.getByRole("button", { name: "Human" });
    const customButton = view.getByRole("button", { name: "Custom" });
    expect(view.getByText("No race traits.")).toBeTruthy();
    fireEvent.focus(customButton);
    expect(view.getByText("Preview")).toBeTruthy();
    expect(customButton.getAttribute("aria-pressed")).toBe("false");
    fireEvent.mouseEnter(humanButton);
    expect(view.getByText("No race traits.")).toBeTruthy();
    fireEvent.click(customButton);
    expect(customButton.getAttribute("aria-pressed")).toBe("true");
    expect(view.getByText("Selected race")).toBeTruthy();
  });
});
