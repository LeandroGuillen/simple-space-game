import { attachModalBehavior } from "../lib/modal";
import { Portal } from "solid-js/web";
import { For, Show, createEffect, createMemo, createSignal } from "solid-js";

import type { ShipDesignForm, ShipDesignViewModel } from "../lib/types/domain";
import { selectedWeaponName, withWeapon } from "../lib/ship-weapons";
import { defaultShipDesignForm } from "../lib/utils";
import { useAppContext } from "../state/app-context";

export type ShipDesignModalMode = "create" | "edit" | "duplicate";

type ShipDesignModalProps = {
  isOpen: boolean;
  mode: ShipDesignModalMode;
  initial?: ShipDesignViewModel | null;
  onClose: () => void;
};

const EYEBROW_BY_MODE: Record<ShipDesignModalMode, string> = {
  create: "New Design",
  edit: "Edit Design",
  duplicate: "Duplicate Design",
};

export function ShipDesignModal(props: ShipDesignModalProps) {
  const { state, saveShipDesign, suggestShipDesign } = useAppContext();
  const [suggestError, setSuggestError] = createSignal("");
  const [suggestMessage, setSuggestMessage] = createSignal("");
  const [isSuggesting, setIsSuggesting] = createSignal(false);
  const parts = createMemo(() => state.game?.resources.shipParts);
  const beams = createMemo(() => {
    const researched = state.game?.players.find((p) => p.id === state.session.playerId)?.available.beams ?? [];
    return (parts()?.beams ?? []).filter((beam) => beam.maxDamage > 0 && researched.includes(beam.name));
  });
  const missiles = createMemo(() => {
    const researched = state.game?.players.find((p) => p.id === state.session.playerId)?.available.missiles ?? [];
    return (parts()?.missiles ?? []).filter((missile) => missile.damage > 0 && researched.includes(missile.name));
  });
  const [form, setForm] = createSignal<ShipDesignForm>(
    defaultShipDesignForm(state.game),
  );

  createEffect(() => {
    if (!props.isOpen) {
      return;
    }
    const base = defaultShipDesignForm(state.game);
    const source = props.initial;
    setSuggestError("");
    setSuggestMessage("");
    if (!source) {
      setForm({ ...base, name: "" });
      return;
    }
    setForm({
      name: props.mode === "duplicate" ? `${source.name} Copy` : source.name,
      drive: source.drive.name,
      beam: source.beam ?? "",
      missile: source.missile ?? "",
      computer: source.computer.name,
      shield: source.shield.name,
      armor: source.armor.name,
      hullSize: source.hullSize.name,
      fuelCell: source.fuelCell.name,
    });
  });

  // Fills the form with a server draft; nothing is saved until "Save Design".
  // On failure the current draft stays as it was.
  const suggest = async () => {
    setIsSuggesting(true);
    setSuggestError("");
    setSuggestMessage("");
    try {
      const suggestion = await suggestShipDesign();
      setForm(suggestion);
      setSuggestMessage("Suggested design loaded. Review and edit it, then save.");
    } catch (error) {
      setSuggestError(error instanceof Error ? error.message : "Unable to suggest a ship design.");
    } finally {
      setIsSuggesting(false);
    }
  };

  const submit = async (event: SubmitEvent) => {
    event.preventDefault();
    const apiMode = props.mode === "edit" ? "update" : "create";
    try {
      await saveShipDesign(form(), apiMode);
      props.onClose();
    } catch {
      // error surfaced via app context state
    }
  };

  return (
    <Show when={props.isOpen}>
      <Portal>
        <div class="research-modal-backdrop" onClick={props.onClose}>
          <section
            class="research-modal panel ship-design-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="ship-design-modal-title"
            ref={(element) => attachModalBehavior(element, props.onClose)}
            onClick={(event) => event.stopPropagation()}
          >
            <div class="research-modal-header">
              <div>
                <p class="eyebrow">{EYEBROW_BY_MODE[props.mode]}</p>
                <h2 id="ship-design-modal-title">Design Ship</h2>
              </div>
              <button
                class="secondary-button research-close-button"
                type="button"
                onClick={props.onClose}
              >
                Close
              </button>
            </div>

            <form class="stack-form" onSubmit={submit}>
              <Show when={props.mode !== "edit"}>
                <div class="ship-design-suggest-row">
                  <button
                    class="secondary-button"
                    type="button"
                    onClick={() => void suggest()}
                    disabled={isSuggesting() || state.isBusy}
                  >
                    {isSuggesting() ? "Suggesting…" : "Suggest design"}
                  </button>
                  <span class="ship-design-suggest-hint">Fill the form from your best researched parts.</span>
                </div>
                <Show when={suggestError()}>
                  <p class="ship-design-suggest-error" role="alert">{suggestError()}</p>
                </Show>
                <Show when={suggestMessage()}>
                  <p class="ship-design-suggest-message" role="status">{suggestMessage()}</p>
                </Show>
              </Show>
              <label class="field">
                <span>Design Name</span>
                <input
                  value={form().name}
                  onInput={(event) =>
                    setForm({ ...form(), name: event.currentTarget.value })
                  }
                  disabled={props.mode === "edit"}
                  required
                />
              </label>

              <Show when={parts()}>
                {(shipParts) => (
                  <div class="ship-design-options">
                    <OptionToggleField
                      label="Weapon (None = utility ship, cannot fight)"
                      value={selectedWeaponName(form()) || "None"}
                      options={[
                        "None",
                        ...beams().map((beam) => beam.name),
                        ...missiles().map((missile) => missile.name),
                      ]}
                      onSelect={(value) =>
                        setForm(withWeapon(form(), value === "None" ? "" : value, missiles()))
                      }
                    />
                    <OptionToggleField
                      label="Hull"
                      value={form().hullSize}
                      options={shipParts().hullSizes.map((item) => item.name)}
                      onSelect={(value) =>
                        setForm({ ...form(), hullSize: value })
                      }
                    />
                    <OptionToggleField
                      label="Drive"
                      value={form().drive}
                      options={shipParts().drives.map((item) => item.name)}
                      onSelect={(value) =>
                        setForm({ ...form(), drive: value })
                      }
                    />
                    <OptionToggleField
                      label="Computer"
                      value={form().computer}
                      options={shipParts().computers.map((item) => item.name)}
                      onSelect={(value) =>
                        setForm({ ...form(), computer: value })
                      }
                    />
                    <OptionToggleField
                      label="Shield"
                      value={form().shield}
                      options={shipParts().shields.map((item) => item.name)}
                      onSelect={(value) =>
                        setForm({ ...form(), shield: value })
                      }
                    />
                    <OptionToggleField
                      label="Armor"
                      value={form().armor}
                      options={shipParts().armors.map((item) => item.name)}
                      onSelect={(value) =>
                        setForm({ ...form(), armor: value })
                      }
                    />
                    <OptionToggleField
                      label="Fuel"
                      value={form().fuelCell}
                      options={shipParts().fuelCells.map((item) => item.name)}
                      onSelect={(value) =>
                        setForm({ ...form(), fuelCell: value })
                      }
                    />
                  </div>
                )}
              </Show>

              <div class="ship-design-modal-actions">
                <button
                  class="secondary-button"
                  type="button"
                  onClick={props.onClose}
                >
                  Cancel
                </button>
                <button
                  class="primary-button"
                  type="submit"
                  disabled={state.isBusy}
                >
                  {props.mode === "edit" ? "Save Changes" : "Save Design"}
                </button>
              </div>
            </form>
          </section>
        </div>
      </Portal>
    </Show>
  );
}

export function OptionToggleField(props: {
  label: string;
  value: string;
  options: string[];
  onSelect: (value: string) => void;
  disabled?: boolean;
}) {
  return (
    <div class="option-row">
      <span class="option-label">{props.label}</span>
      <div class="option-toggle-group" role="group" aria-label={props.label}>
        <For each={props.options}>
          {(option) => (
            <button
              type="button"
              class="option-toggle"
              classList={{ selected: props.value === option }}
              aria-pressed={props.value === option}
              disabled={props.disabled}
              onClick={() => !props.disabled && props.onSelect(option)}
            >
              {option}
            </button>
          )}
        </For>
      </div>
    </div>
  );
}
