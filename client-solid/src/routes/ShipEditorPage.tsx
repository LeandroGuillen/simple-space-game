import { useNavigate } from "@solidjs/router";
import type { Accessor } from "solid-js";
import { createEffect, createMemo, createSignal, For, onCleanup, onMount, Show } from "solid-js";

import { ApiClient } from "../lib/api";
import {
  getCustomShipDesignDraft,
  setCustomShipDesignDraft,
} from "../lib/custom-ship-design-storage";
import type {
  ApiArmor,
  ApiComputer,
  ApiDrive,
  ApiFuelCell,
  ApiHullSize,
  ApiShield,
  ApiShipParts,
  ShipDesignTemplateRow,
} from "../lib/types/api";
import type { ShipDesignForm } from "../lib/types/domain";
import { defaultShipDesignFormFromParts } from "../lib/utils";
import { useAppContext } from "../state/app-context";

function BackArrowIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true">
      <path
        d="M11 14L5 9l6-5"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}

function templateRowToForm(row: ShipDesignTemplateRow): ShipDesignForm {
  return {
    name: row.name,
    beam: row.beam ?? "",
    drive: row.drive,
    computer: row.computer,
    shield: row.shield,
    armor: row.armor,
    hullSize: row.hull_size,
    fuelCell: row.fuel_cell,
  };
}

function driveLines(d: ApiDrive): { k: string; v: string }[] {
  return [
    { k: "Parsecs / turn", v: String(d.interstellar_speed) },
    { k: "Combat speed", v: String(d.combat_speed) },
  ];
}

function armorLines(a: ApiArmor): { k: string; v: string }[] {
  return [
    { k: "Armor multiplier", v: String(a.multiplier) },
    { k: "Troop strength", v: String(a.ground_troop_strength) },
  ];
}

function shieldLines(s: ApiShield): { k: string; v: string }[] {
  return [
    { k: "Recharge", v: s.recharge_rate.toFixed(2) },
    { k: "Damage blocked", v: String(s.reduce_damage) },
    { k: "Absorption × hull HP", v: String(s.absortion_factor) },
  ];
}

function computerLines(c: ApiComputer): { k: string; v: string }[] {
  return [{ k: "Beam attack", v: `+${c.beam_attack_bonus}` }];
}

function fuelLines(f: ApiFuelCell): { k: string; v: string }[] {
  return [{ k: "Fuel range", v: `${f.autonomy} parsecs` }];
}

function statOrDash<T>(
  found: T | null | undefined,
  builder: (x: T) => { k: string; v: string }[],
): { k: string; v: string }[] {
  if (found == null) return [{ k: "—", v: "—" }];
  return builder(found);
}

type ShipPartDropdownKey = "drive" | "armor" | "shield" | "computer" | "fuel";

type PartOptionRow = { value: string; subtitle?: string };

function PartPickerPanel(props: {
  title: string;
  dropdownKey: ShipPartDropdownKey;
  openDropdown: Accessor<ShipPartDropdownKey | null>;
  setOpenDropdown: (key: ShipPartDropdownKey | null) => void;
  selectionLabel: string;
  stats: { k: string; v: string }[];
  optionRows: PartOptionRow[];
  value: string;
  onSelect: (v: string) => void;
  disabled?: boolean;
}) {
  const isOpen = () => props.openDropdown() === props.dropdownKey;

  return (
    <div class="ship-editor-part-block">
      <h3 class="ship-editor-part-label">{props.title}</h3>
        <div
          class="ship-editor-part-dd-root"
          classList={{ "is-open": isOpen() }}
          data-ship-editor-dd-root={props.dropdownKey}
        >
          <button
            type="button"
            class="ship-editor-part-trigger"
            aria-expanded={isOpen()}
            aria-haspopup="listbox"
            disabled={props.disabled}
            onClick={(e) => {
              e.stopPropagation();
              if (props.disabled) return;
              props.setOpenDropdown(isOpen() ? null : props.dropdownKey);
            }}
          >
            <span class="ship-editor-part-trigger-label">{props.selectionLabel}</span>
            <svg
              class="ship-editor-part-trigger-chevron"
              width="14"
              height="14"
              viewBox="0 0 14 14"
              aria-hidden="true"
            >
              <path
                d="M3.5 5.25L7 8.75l3.5-3.5"
                fill="none"
                stroke="currentColor"
                stroke-width="1.4"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
          <Show when={isOpen()}>
            <ul class="ship-editor-part-dropdown" role="listbox" aria-label={`${props.title} choices`}>
              <For each={props.optionRows}>
                {(row) => (
                  <li role="none">
                    <button
                      type="button"
                      role="option"
                      class="ship-editor-part-dropdown-option"
                      classList={{ "is-current": props.value === row.value }}
                      aria-selected={props.value === row.value}
                      onClick={(e) => {
                        e.stopPropagation();
                        props.onSelect(row.value);
                        props.setOpenDropdown(null);
                      }}
                    >
                      <span class="ship-editor-part-dropdown-name">{row.value}</span>
                      {row.subtitle ? (
                        <span class="ship-editor-part-dropdown-sub">{row.subtitle}</span>
                      ) : null}
                    </button>
                  </li>
                )}
              </For>
            </ul>
          </Show>
        </div>
      <ul class="ship-editor-part-stats">
        <For each={props.stats}>{(s) => (
          <li>
            <span>{s.k}</span>
            <span>{s.v}</span>
          </li>
        )}</For>
      </ul>
    </div>
  );
}

export function ShipEditorPage() {
  const navigate = useNavigate();
  const { state } = useAppContext();
  const [allDesigns, setAllDesigns] = createSignal<ShipDesignTemplateRow[]>([]);
  const [parts, setParts] = createSignal<ApiShipParts | null>(null);
  const beams = createMemo(() => (parts()?.beams ?? []).filter((beam) => beam.maxDamage > 0));
  const [listFocus, setListFocus] = createSignal<string | null>(null);
  const [isDraft, setIsDraft] = createSignal(false);
  const [form, setForm] = createSignal<ShipDesignForm>(defaultShipDesignFormFromParts(null));
  const [saveError, setSaveError] = createSignal("");
  const [saveMessage, setSaveMessage] = createSignal("");
  const [confirmDelete, setConfirmDelete] = createSignal("");
  const [isSaving, setIsSaving] = createSignal(false);

  const client = createMemo(() =>
    state.session.serverUrl ? new ApiClient(state.session.serverUrl, state.session.playerId) : null,
  );

  const readOnly = createMemo(() => {
    if (isDraft()) return false;
    const focus = listFocus();
    if (!focus) return false;
    const row = allDesigns().find((d) => d.name === focus);
    return Boolean(row?.built_in);
  });

  const [openPartDropdown, setOpenPartDropdown] = createSignal<ShipPartDropdownKey | null>(null);

  createEffect(() => {
    if (openPartDropdown() === null) return;
    const onDoc = (e: MouseEvent) => {
      const open = openPartDropdown();
      if (open === null) return;
      const el = e.target as HTMLElement | null;
      const hit = el?.closest("[data-ship-editor-dd-root]");
      const key = hit?.getAttribute("data-ship-editor-dd-root");
      if (key === open) return;
      setOpenPartDropdown(null);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenPartDropdown(null);
    };
    queueMicrotask(() => {
      document.addEventListener("click", onDoc);
      document.addEventListener("keydown", onKey);
    });
    onCleanup(() => {
      document.removeEventListener("click", onDoc);
      document.removeEventListener("keydown", onKey);
    });
  });

  createEffect(() => {
    if (readOnly()) setOpenPartDropdown(null);
  });

  const showSaveAsNew = createMemo(() => {
    if (readOnly() || !client() || isDraft()) return false;
    const focus = listFocus();
    if (!focus) return false;
    const row = allDesigns().find((d) => d.name === focus);
    if (!row || row.built_in) return false;
    return form().name.trim() !== focus && form().name.trim().length > 0;
  });

  const errors = createMemo(() => {
    const e: string[] = [];
    const f = form();
    if (!f.name.trim()) {
      e.push("Design name is required.");
    }
    if (!parts()) {
      e.push("Ship parts did not load.");
    }
    if (parts() && !parts()?.drives.some((d) => d.name === f.drive && d.interstellar_speed > 0)) e.push("Choose an interstellar drive.");
    if (parts() && !parts()?.fuel_cells.some((cell) => cell.name === f.fuelCell && cell.autonomy > 0)) e.push("Choose a fuel cell with range.");
    if (f.beam && !parts()?.beams?.some((b) => b.name === f.beam)) e.push("Choose a supported weapon.");
    return e;
  });

  const warnings = createMemo(() => {
    const w: string[] = [];
    if (form().hullSize && form().hullSize !== "Frigate") {
      w.push("Only Frigate hulls currently have colony production rules in-game.");
    }
    return w;
  });

  const hullSel = createMemo(() => {
    const p = parts();
    const n = form().hullSize;
    return p?.hull_sizes.find((x) => x.name === n) ?? null;
  });

  const driveSel = createMemo(() => {
    const p = parts();
    const n = form().drive;
    return p?.drives.find((x) => x.name === n) ?? null;
  });

  const armorSel = createMemo(() => {
    const p = parts();
    const n = form().armor;
    return p?.armors.find((x) => x.name === n) ?? null;
  });

  const shieldSel = createMemo(() => {
    const p = parts();
    const n = form().shield;
    return p?.shields.find((x) => x.name === n) ?? null;
  });

  const computerSel = createMemo(() => {
    const p = parts();
    const n = form().computer;
    return p?.computers.find((x) => x.name === n) ?? null;
  });

  const fuelSel = createMemo(() => {
    const p = parts();
    const n = form().fuelCell;
    return p?.fuel_cells.find((x) => x.name === n) ?? null;
  });

  const loadFormFromTemplate = (row: ShipDesignTemplateRow) => {
    setForm(templateRowToForm(row));
  };

  const refreshAll = async (): Promise<ShipDesignTemplateRow[]> => {
    if (!client()) return [];
    try {
      const rows = await client()!.getShipDesignTemplates();
      setAllDesigns(rows);
      return rows;
    } catch {
      setSaveError("Unable to load ship designs.");
      return [];
    }
  };

  const refreshParts = async () => {
    if (!client()) return;
    try {
      const fetched = await client()!.getShipParts();
      setParts(fetched);
      setForm((prev) =>
        prev.name.trim()
          ? prev
          : { ...defaultShipDesignFormFromParts(fetched), name: prev.name },
      );
    } catch {
      setSaveError("Unable to load ship parts.");
      setParts(null);
    }
  };

  onMount(() => {
    void (async () => {
      await refreshParts();
      const rows = await refreshAll();
      const draft = getCustomShipDesignDraft();
      if (draft?.name.trim()) {
        setIsDraft(true);
        setListFocus(null);
        setForm(draft);
        return;
      }
      const firstBuiltin = rows.find((r) => r.built_in);
      if (firstBuiltin) {
        setListFocus(firstBuiltin.name);
        setIsDraft(false);
        loadFormFromTemplate(firstBuiltin);
      }
    })();
  });

  const selectRow = (row: ShipDesignTemplateRow) => {
    setOpenPartDropdown(null);
    setListFocus(row.name);
    setIsDraft(false);
    loadFormFromTemplate(row);
    setSaveError("");
    setSaveMessage("");
    setConfirmDelete("");
  };

  const handleNew = () => {
    const taken = new Set(allDesigns().map((d) => d.name));
    let candidate = "Custom Ship";
    if (taken.has(candidate)) {
      for (let i = 2; i < 100; i += 1) {
        const next = `Custom Ship ${i}`;
        if (!taken.has(next)) {
          candidate = next;
          break;
        }
      }
    }
    setIsDraft(true);
    setListFocus(null);
    setOpenPartDropdown(null);
    setForm({ ...defaultShipDesignFormFromParts(parts()), name: candidate });
    setSaveError("");
    setSaveMessage("");
    setConfirmDelete("");
  };

  const handleCopy = () => {
    const next = nextCopyName(form().name.trim() || "Design", allDesigns());
    setIsDraft(true);
    setListFocus(null);
    setOpenPartDropdown(null);
    setForm({ ...form(), name: next });
    setSaveError("");
    setSaveMessage("");
    setConfirmDelete("");
  };

  const handleClearComponents = () => {
    const p = parts();
    if (!p || readOnly()) return;
    setOpenPartDropdown(null);
    setForm((prev) => ({
      ...defaultShipDesignFormFromParts(p),
      name: prev.name,
    }));
  };

  const saveCurrentDesign = async (opts?: { renameFrom?: string; saveAsNew?: boolean }) => {
    if (readOnly()) return null;
    setIsSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      const saved = client()
        ? await client()!.saveCustomShipDesign(form(), opts)
        : null;
      if (saved) {
        const asForm = templateRowToForm(saved);
        setCustomShipDesignDraft(asForm);
        setSaveMessage(`Saved ${saved.name}.`);
        setIsDraft(false);
        setListFocus(saved.name);
      }
      await refreshAll();
      return saved;
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to save ship design.");
      return null;
    } finally {
      setIsSaving(false);
    }
  };

  const handleSave = async () => {
    if (errors().length > 0 || readOnly()) return;
    const focus = listFocus();
    const trimmed = form().name.trim();
    const row = focus ? allDesigns().find((d) => d.name === focus) : undefined;
    const renaming = Boolean(!isDraft() && focus && row && !row.built_in && trimmed !== focus);
    await saveCurrentDesign(renaming ? { renameFrom: focus! } : undefined);
  };

  const handleSaveAsNew = async () => {
    if (errors().length > 0 || readOnly() || !showSaveAsNew()) return;
    await saveCurrentDesign({ saveAsNew: true });
  };

  const handleDelete = async () => {
    if (readOnly() || isDraft()) return;
    const nm = listFocus();
    if (!nm) return;
    const row = allDesigns().find((d) => d.name === nm);
    if (!row || row.built_in) return;
    if (confirmDelete() !== nm) {
      setConfirmDelete(nm);
      return;
    }
    if (!client()) return;
    setIsSaving(true);
    setSaveError("");
    setSaveMessage("");
    try {
      await client()!.deleteCustomShipDesign(nm);
      setConfirmDelete("");
      const updated = await refreshAll();
      const builtin = updated.find((r) => r.built_in);
      if (builtin) {
        selectRow(builtin);
      } else {
        handleNew();
      }
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : "Unable to delete ship design.");
    } finally {
      setIsSaving(false);
    }
  };

  const rowSelected = (row: ShipDesignTemplateRow) => !isDraft() && listFocus() === row.name;

  const updateForm = (patch: Partial<ShipDesignForm>) => {
    setForm({ ...form(), ...patch });
  };

  return (
    <section class="race-editor-page ship-editor-page">
      <div class="ship-editor-frame">
        <header class="ship-editor-header">
          <div class="ship-editor-header-primary">
            <button
              class="race-editor-back"
              type="button"
              onClick={() => navigate("/")}
              aria-label="Back to home"
            >
              <BackArrowIcon />
            </button>
          <label class="ship-editor-name-bar">
            <span>Designation</span>
            <input
              value={form().name}
              maxlength={48}
              placeholder="Ship name"
              disabled={readOnly()}
              onInput={(event) => updateForm({ name: event.currentTarget.value })}
            />
          </label>
          <h1 class="ship-editor-display-title">Ship design</h1>
          </div>
          <div class="ship-editor-header-bar" aria-label="Ship design actions">
            <div class="ship-editor-header-stats">
            <div>
              <span>Cost</span>
              <strong>—</strong>
            </div>
            <div>
              <span>Space Available</span>
              <strong>—</strong>
            </div>
          </div>
            <div class="ship-editor-header-actions">
              <button class="link-button" type="button" onClick={handleNew} disabled={!client() || isSaving()}>
              New
            </button>
            <button class="link-button" type="button" onClick={handleCopy} disabled={!client() || isSaving()}>
              Copy
            </button>
            <button
              class="secondary-button"
              type="button"
              onClick={handleClearComponents}
              disabled={readOnly() || !parts() || isSaving()}
            >
              Clear fit-out
            </button>
            <Show when={showSaveAsNew()}>
              <button
                class="secondary-button"
                type="button"
                onClick={() => void handleSaveAsNew()}
                disabled={errors().length > 0 || readOnly() || isSaving()}
              >
                Save as new
              </button>
            </Show>
            <button
              class="link-button"
              type="button"
              classList={{ "race-editor-delete": true }}
              onClick={() => void handleDelete()}
              disabled={readOnly() || isDraft() || isSaving() || !listFocus()}
            >
              {confirmDelete() === listFocus() ? "Confirm delete" : "Delete"}
            </button>
            <button class="link-button" type="button" onClick={() => navigate("/")} disabled={isSaving()}>
              Cancel
            </button>
            <button
              class="primary-button"
              type="button"
              onClick={() => void handleSave()}
              disabled={errors().length > 0 || readOnly() || isSaving()}
            >
              Save design
            </button>
            </div>
          </div>
        </header>
        <Show when={saveError()}>
          <p class="race-editor-save-error" role="alert">{saveError()}</p>
        </Show>
        <p class="ship-editor-hint">Designs used by construction, build queues, or existing ships cannot be changed or deleted. Copy the design and save it under a new name.</p>

        <div class="ship-editor-card">
          <aside class="ship-editor-sidebar" aria-label="Saved ship designs">
            <h2 class="ship-editor-sidebar-title">Fleet roster</h2>
            <div class="ship-editor-sidebar-list">
              <Show when={isDraft()}>
                <div class="race-editor-draft-pill">Unsaved draft</div>
              </Show>
              <For each={allDesigns()}>
                {(row) => (
                  <button
                    type="button"
                    class="race-editor-option construction-project-option"
                    classList={{ "is-selected": rowSelected(row) }}
                    onClick={() => selectRow(row)}
                  >
                    <span class="race-editor-option-marker" />
                    <span class="race-editor-option-main">
                      <span class="race-editor-option-name">
                        {row.name}
                        {row.built_in ? <span class="race-editor-badge">Built-in</span> : null}
                      </span>
                      <span class="race-editor-option-meta">
                        {row.hull_size} · {row.drive}
                      </span>
                    </span>
                  </button>
                )}
              </For>
            </div>
          </aside>

          <div
            class="race-editor-main ship-editor-main"
            classList={{ "race-editor-main--readonly": readOnly() }}
          >
            <Show when={parts()}>
              {(shipParts) => (
                <>
                  <div class="ship-editor-workstation">
                    <div class="ship-editor-hull-column">
                      <p class="ship-editor-hull-heading">Ship class</p>
                      <div class="ship-editor-hull-space">
                        <span>Hull space</span>
                        <strong>{hullSel()?.space ?? "—"}</strong>
                      </div>
                      <div class="ship-editor-class-list" role="listbox" aria-label="Hull sizes">
                        <For each={shipParts().hull_sizes}>
                          {(h: ApiHullSize) => (
                            <button
                              type="button"
                              role="option"
                              aria-selected={form().hullSize === h.name}
                              class="ship-editor-class-btn"
                              classList={{ "is-active": form().hullSize === h.name }}
                              disabled={readOnly()}
                              onClick={() => updateForm({ hullSize: h.name })}
                            >
                              {h.name}
                            </button>
                          )}
                        </For>
                      </div>
                    </div>

                    <div class="ship-editor-parts-grid">
                      <PartPickerPanel
                        title="Drive"
                        dropdownKey="drive"
                        openDropdown={openPartDropdown}
                        setOpenDropdown={setOpenPartDropdown}
                        selectionLabel={(driveSel()?.name ?? form().drive) || "—"}
                        stats={statOrDash(driveSel(), driveLines)}
                        optionRows={shipParts().drives.map((d: ApiDrive) => ({
                          value: d.name,
                          subtitle: `${d.interstellar_speed} pc/turn · combat ${d.combat_speed}`,
                        }))}
                        value={form().drive}
                        onSelect={(v) => updateForm({ drive: v })}
                        disabled={readOnly()}
                      />
                      <PartPickerPanel
                        title="Shield"
                        dropdownKey="shield"
                        openDropdown={openPartDropdown}
                        setOpenDropdown={setOpenPartDropdown}
                        selectionLabel={(shieldSel()?.name ?? form().shield) || "—"}
                        stats={statOrDash(shieldSel(), shieldLines)}
                        optionRows={shipParts().shields.map((s: ApiShield) => ({
                          value: s.name,
                          subtitle: `recharge ${s.recharge_rate.toFixed(1)} · block ${s.reduce_damage}`,
                        }))}
                        value={form().shield}
                        onSelect={(v) => updateForm({ shield: v })}
                        disabled={readOnly()}
                      />
                      <PartPickerPanel
                        title="Armor"
                        dropdownKey="armor"
                        openDropdown={openPartDropdown}
                        setOpenDropdown={setOpenPartDropdown}
                        selectionLabel={(armorSel()?.name ?? form().armor) || "—"}
                        stats={statOrDash(armorSel(), armorLines)}
                        optionRows={shipParts().armors.map((a: ApiArmor) => ({
                          value: a.name,
                          subtitle: `×${a.multiplier} structure · troops ${a.ground_troop_strength}`,
                        }))}
                        value={form().armor}
                        onSelect={(v) => updateForm({ armor: v })}
                        disabled={readOnly()}
                      />
                      <PartPickerPanel
                        title="Computer"
                        dropdownKey="computer"
                        openDropdown={openPartDropdown}
                        setOpenDropdown={setOpenPartDropdown}
                        selectionLabel={(computerSel()?.name ?? form().computer) || "—"}
                        stats={statOrDash(computerSel(), computerLines)}
                        optionRows={shipParts().computers.map((c: ApiComputer) => ({
                          value: c.name,
                          subtitle: `beam attack +${c.beam_attack_bonus}`,
                        }))}
                        value={form().computer}
                        onSelect={(v) => updateForm({ computer: v })}
                        disabled={readOnly()}
                      />
                      <PartPickerPanel
                        title="Fuel cell"
                        dropdownKey="fuel"
                        openDropdown={openPartDropdown}
                        setOpenDropdown={setOpenPartDropdown}
                        selectionLabel={(fuelSel()?.name ?? form().fuelCell) || "—"}
                        stats={statOrDash(fuelSel(), fuelLines)}
                        optionRows={shipParts().fuel_cells.map((f: ApiFuelCell) => ({
                          value: f.name,
                          subtitle: `${f.autonomy} parsecs`,
                        }))}
                        value={form().fuelCell}
                        onSelect={(v) => updateForm({ fuelCell: v })}
                        disabled={readOnly()}
                      />
                    </div>

                    <section class="ship-editor-bay" aria-label="Weapons">
                      <h3 class="ship-editor-bay-title">Weapons</h3>
                      <div class="ship-editor-bay-table-wrap">
                        <table class="ship-editor-data-table">
                          <thead>
                            <tr>
                              <th scope="col">Weapon</th>
                              <th scope="col">Damage</th>
                              <th scope="col">Actions</th>
                            </tr>
                          </thead>
                          <tbody>
                            <Show when={form().beam}>
                              <tr>
                                <td>
                                  <select
                                    aria-label="Weapon"
                                    value={form().beam}
                                    disabled={readOnly()}
                                    onChange={(e) => updateForm({ beam: e.currentTarget.value })}
                                  >
                                    <For each={beams()}>
                                      {(b) => <option value={b.name}>{b.name}</option>}
                                    </For>
                                  </select>
                                </td>
                                <td>
                                  {parts()?.beams?.find((b) => b.name === form().beam)?.minDamage}
                                  –{parts()?.beams?.find((b) => b.name === form().beam)?.maxDamage}
                                </td>
                                <td>
                                  <button type="button" disabled={readOnly()} onClick={() => updateForm({ beam: "" })}>
                                    Remove weapon
                                  </button>
                                </td>
                              </tr>
                            </Show>
                          </tbody>
                        </table>
                      </div>
                      <button
                        type="button"
                        class="ship-editor-bay-add-hint"
                        disabled={readOnly() || !!form().beam || !beams().length}
                        onClick={() => updateForm({ beam: beams()[0]?.name ?? "" })}
                      >
                        + Add weapon
                      </button>
                      <p>
                        {form().beam
                          ? "Armed design. Requires researched weapon technology to build."
                          : "Weaponless utility design — cannot fight. Add a weapon to create a combat ship."}
                      </p>
                    </section>

                    <section class="ship-editor-bay" aria-label="Special systems">
                      <h3 class="ship-editor-bay-title">Special systems</h3>
                      <div class="ship-editor-bay-table-wrap">
                      <table class="ship-editor-data-table ship-editor-data-table--systems">
                        <thead>
                          <tr>
                            <th scope="col">System</th>
                            <th scope="col">Description</th>
                          </tr>
                        </thead>
                        <tbody />
                      </table>
                      </div>
                      <p class="ship-editor-bay-add-hint">+ Add system</p>
                    </section>

                    <div class="ship-editor-alerts race-editor-messages">
                      <Show when={errors().length > 0}>
                        <ul class="race-editor-errors">
                          <For each={errors()}>{(error) => <li>{error}</li>}</For>
                        </ul>
                      </Show>
                      <Show when={errors().length === 0 && warnings().length > 0}>
                        <ul class="race-editor-warnings">
                          <For each={warnings()}>{(w) => <li>Note: {w}</li>}</For>
                        </ul>
                      </Show>
                      <Show when={readOnly()}>
                        <p class="race-editor-readonly-note">
                          Built-in designs are read-only. Use Copy to derive an editable template.
                        </p>
                      </Show>
                      <Show when={saveMessage()}>
                        <p class="race-editor-save-message">{saveMessage()}</p>
                      </Show>
                    </div>
                  </div>
                </>
              )}
            </Show>

            <Show when={!parts()}>
              <div class="race-editor-messages">
                <Show when={errors().length > 0}>
                  <ul class="race-editor-errors">
                    <For each={errors()}>{(error) => <li>{error}</li>}</For>
                  </ul>
                </Show>
              </div>
            </Show>
          </div>
        </div>
      </div>
    </section>
  );
}

function nextCopyName(baseName: string, designs: ShipDesignTemplateRow[]) {
  const existing = new Set(designs.map((d) => d.name));
  const base = `${baseName} Copy`;
  if (!existing.has(base)) return base;
  for (let index = 2; index < 100; index += 1) {
    const candidate = `${base} ${index}`;
    if (!existing.has(candidate)) return candidate;
  }
  return base;
}
