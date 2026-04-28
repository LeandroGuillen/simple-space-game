import {
  createEffect,
  createMemo,
  createSignal,
  onMount,
  Show,
} from "solid-js";

import type { ShipDesignForm } from "../lib/types/domain";
import { defaultShipDesignForm } from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function FleetsPage() {
  const { state, currentPlayer, saveShipDesign } = useAppContext();
  const [form, setForm] = createSignal<ShipDesignForm>(
    defaultShipDesignForm(state.game),
  );

  onMount(() => {
  });

  createEffect(() => {
    if (state.game && !form().drive) {
      setForm(defaultShipDesignForm(state.game));
    }
  });

  const commonDesigns = createMemo(
    () => state.game?.resources.shipDesignCollection.shipDesigns ?? [],
  );
  const playerDesigns = createMemo(() => currentPlayer()?.shipDesigns ?? []);
  const parts = createMemo(() => state.game?.resources.shipParts);

  function loadIntoForm(name: string) {
    const source = [...playerDesigns(), ...commonDesigns()].find(
      (design) => design.name === name,
    );
    if (!source) {
      return;
    }
    setForm({
      name: source.name,
      drive: source.drive.name,
      computer: source.computer.name,
      shield: source.shield.name,
      armor: source.armor.name,
      hullSize: source.hullSize.name,
      fuelCell: source.fuelCell.name,
    });
  }

  const submit = async (event: SubmitEvent) => {
    event.preventDefault();
    const exists = playerDesigns().some(
      (design) => design.name === form().name,
    );
    await saveShipDesign(form(), exists ? "update" : "create");
  };

  return (
    <section class="page-grid">
      <section class="panel">
        <p class="eyebrow">Ship Designs</p>
        <h2>Fleet architecture</h2>
        <div class="table-frame">
          <table class="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Hull</th>
                <th>Drive</th>
                <th>Computer</th>
                <th>Shield</th>
                <th>Fuel</th>
                <th>Type</th>
              </tr>
            </thead>
            <tbody>
              {[...playerDesigns(), ...commonDesigns()].map((design) => (
                <tr onClick={() => loadIntoForm(design.name)}>
                  <td>{design.name}</td>
                  <td>{design.hullSize.name}</td>
                  <td>{design.drive.name}</td>
                  <td>{design.computer.name}</td>
                  <td>{design.shield.name}</td>
                  <td>{design.fuelCell.name}</td>
                  <td>{design.isCommon ? "Common" : "Custom"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div class="table-frame">
          <p class="eyebrow">Fleet Table</p>
          <table class="data-table">
            <thead>
              <tr>
                <th>Fleet ID</th>
                <th>Commander</th>
              </tr>
            </thead>
            <tbody>
              {currentPlayer()?.fleets.map((fleetId) => (
                <tr>
                  <td>{fleetId}</td>
                  <td>{currentPlayer()?.name}</td>
                </tr>
              ))}
              <Show when={(currentPlayer()?.fleets.length ?? 0) === 0}>
                <tr>
                  <td colSpan={2}>
                    No fleet records available for the current commander.
                  </td>
                </tr>
              </Show>
            </tbody>
          </table>
        </div>

        <div class="fleet-note">
          <p class="eyebrow">Fleet Manifest</p>
          <p>
            The backend currently exposes fleet IDs but not a full fleet object
            collection. This screen focuses on ship design operations until
            fleet payloads are expanded.
          </p>
        </div>
      </section>

      <section class="panel">
        <p class="eyebrow">Design Console</p>
        <form class="stack-form" onSubmit={submit}>
          <label class="field">
            <span>Design Name</span>
            <input
              value={form().name}
              onInput={(event) =>
                setForm({ ...form(), name: event.currentTarget.value })
              }
              required
            />
          </label>

          <Show when={parts()}>
            {(shipParts) => (
              <>
                <SelectField
                  label="Hull Size"
                  value={form().hullSize}
                  options={shipParts().hullSizes.map((item) => item.name)}
                  onChange={(value) => setForm({ ...form(), hullSize: value })}
                />
                <SelectField
                  label="Drive"
                  value={form().drive}
                  options={shipParts().drives.map((item) => item.name)}
                  onChange={(value) => setForm({ ...form(), drive: value })}
                />
                <SelectField
                  label="Computer"
                  value={form().computer}
                  options={shipParts().computers.map((item) => item.name)}
                  onChange={(value) => setForm({ ...form(), computer: value })}
                />
                <SelectField
                  label="Shield"
                  value={form().shield}
                  options={shipParts().shields.map((item) => item.name)}
                  onChange={(value) => setForm({ ...form(), shield: value })}
                />
                <SelectField
                  label="Armor"
                  value={form().armor}
                  options={shipParts().armors.map((item) => item.name)}
                  onChange={(value) => setForm({ ...form(), armor: value })}
                />
                <SelectField
                  label="Fuel Cell"
                  value={form().fuelCell}
                  options={shipParts().fuelCells.map((item) => item.name)}
                  onChange={(value) => setForm({ ...form(), fuelCell: value })}
                />
              </>
            )}
          </Show>

          <button class="primary-button" type="submit" disabled={state.isBusy}>
            Save Design
          </button>
        </form>
      </section>
    </section>
  );
}

function SelectField(props: {
  label: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
}) {
  return (
    <label class="field">
      <span>{props.label}</span>
      <select
        value={props.value}
        onChange={(event) => props.onChange(event.currentTarget.value)}
      >
        {props.options.map((option) => (
          <option value={option}>{option}</option>
        ))}
      </select>
    </label>
  );
}
