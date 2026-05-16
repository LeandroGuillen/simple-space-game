import { Portal } from "solid-js/web";
import { For, Show, createEffect, createMemo, createSignal } from "solid-js";

import type { ShipDesignForm, ShipDesignViewModel } from "../lib/types/domain";
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
  const { state, saveShipDesign } = useAppContext();
  const parts = createMemo(() => state.game?.resources.shipParts);
  const [form, setForm] = createSignal<ShipDesignForm>(
    defaultShipDesignForm(state.game),
  );

  createEffect(() => {
    if (!props.isOpen) {
      return;
    }
    const base = defaultShipDesignForm(state.game);
    const source = props.initial;
    if (!source) {
      setForm({ ...base, name: "" });
      return;
    }
    setForm({
      name: props.mode === "duplicate" ? `${source.name} Copy` : source.name,
      drive: source.drive.name,
      computer: source.computer.name,
      shield: source.shield.name,
      armor: source.armor.name,
      hullSize: source.hullSize.name,
      fuelCell: source.fuelCell.name,
    });
  });

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
