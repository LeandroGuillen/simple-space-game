import { fireEvent, render, cleanup } from "@solidjs/testing-library";
import { afterEach, expect, it, vi } from "vitest";
import { TaxControl } from "../TaxControl";

afterEach(cleanup);

it("offers all six rates and queues the selected rate including zero", () => {
  const change = vi.fn();
  const screen = render(() => <TaxControl rate={.3} onChange={change} />);
  const select = screen.getByLabelText("Empire tax rate") as HTMLSelectElement;
  expect(select.value).toBe("30");
  expect(Array.from(select.options, option => option.text)).toEqual(["0%", "10%", "20%", "30%", "40%", "50%"]);
  fireEvent.change(select, { target: { value: "50" } });
  expect(change).toHaveBeenLastCalledWith(.5);
  fireEvent.change(select, { target: { value: "0" } });
  expect(change).toHaveBeenLastCalledWith(0);
});
