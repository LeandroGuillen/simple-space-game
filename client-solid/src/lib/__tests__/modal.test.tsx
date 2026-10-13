import { cleanup, fireEvent, render } from "@solidjs/testing-library";
import { createSignal, Show } from "solid-js";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { attachModalBehavior } from "../modal";

beforeEach(() => {
  // jsdom does not calculate layout; make controls visible to the focus filter.
  vi.spyOn(HTMLElement.prototype, "getClientRects").mockReturnValue([{}] as unknown as DOMRectList);
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function Dialog(props: { close: () => void }) {
  return <section role="dialog" ref={(element) => attachModalBehavior(element, props.close)}>
    <button>First</button>
    <button disabled>Disabled</button>
    <button>Last</button>
  </section>;
}

describe("dialog behavior", () => {
  it("focuses the dialog, traps Tab in both directions, and restores focus on Escape", async () => {
    const close = vi.fn();
    const view = render(() => {
      const [open, setOpen] = createSignal(false);
      return <>
        <button onClick={() => setOpen(true)}>Open</button>
        <Show when={open()}><Dialog close={() => { close(); setOpen(false); }} /></Show>
      </>;
    });
    const opener = view.getByText("Open");
    opener.focus();
    fireEvent.click(opener);
    await Promise.resolve();
    const first = view.getByText("First");
    const last = view.getByText("Last");
    expect(document.activeElement).toBe(first);
    expect(document.body.style.overflow).toBe("hidden");
    fireEvent.keyDown(first, { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(last);
    fireEvent.keyDown(last, { key: "Tab" });
    expect(document.activeElement).toBe(first);
    fireEvent.keyDown(first, { key: "Escape" });
    expect(close).toHaveBeenCalledOnce();
    expect(view.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(opener);
    expect(document.body.style.overflow).toBe("");
    fireEvent.keyDown(opener, { key: "Escape" });
    expect(close).toHaveBeenCalledOnce();
  });

  it("only dismisses the top dialog and keeps scroll locked until both close", async () => {
    const closeOuter = vi.fn();
    const closeInner = vi.fn();
    const view = render(() => {
      const [outer, setOuter] = createSignal(true);
      const [inner, setInner] = createSignal(true);
      return <>
        <Show when={outer()}><Dialog close={() => { closeOuter(); setOuter(false); }} /></Show>
        <Show when={inner()}><Dialog close={() => { closeInner(); setInner(false); }} /></Show>
      </>;
    });
    await Promise.resolve();
    fireEvent.keyDown(document, { key: "Escape" });
    expect(closeInner).toHaveBeenCalledOnce();
    expect(closeOuter).not.toHaveBeenCalled();
    expect(view.getAllByRole("dialog")).toHaveLength(1);
    expect(view.getByRole("dialog").contains(document.activeElement)).toBe(true);
    expect(document.body.style.overflow).toBe("hidden");
    fireEvent.keyDown(document, { key: "Escape" });
    expect(closeOuter).toHaveBeenCalledOnce();
    expect(document.body.style.overflow).toBe("");
  });
});
