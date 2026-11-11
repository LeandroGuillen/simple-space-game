import { For } from "solid-js";

export function TaxControl(props: { rate: number; onChange: (rate: number) => void }) {
  return (
    <label class="tax-control" title="Converts industry after pollution and Food Replicators to BC, 1:1. Applied next turn.">
      Tax{" "}
      <select aria-label="Empire tax rate" value={Math.round(props.rate * 100)}
        onChange={(event) => props.onChange(Number(event.currentTarget.value) / 100)}>
        <For each={[0, 10, 20, 30, 40, 50]}>{(percent) =>
          <option value={percent}>{percent}%</option>
        }</For>
      </select>
    </label>
  );
}
