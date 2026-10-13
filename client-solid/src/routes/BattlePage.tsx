import { useNavigate, useParams } from "@solidjs/router";
import Phaser from "phaser";
import {
  For,
  Show,
  createEffect,
  createMemo,
  createSignal,
  onCleanup,
  onMount,
} from "solid-js";

import { BattleScene, type BattleIntent } from "../components/battle/BattleScene";
import type {
  ApiTacticalBattle,
  ApiTacticalShip,
  BattleActionRequest,
} from "../lib/types/battle";
import { currentBattleShip } from "../lib/types/battle";
import { useAppContext } from "../state/app-context";

const RETURN_TO_MAP_DELAY_MS = 2200;

type WeaponRow = { label: string; count: number; damage: number };

function weaponRows(ship: ApiTacticalShip | null): WeaponRow[] {
  if (!ship?.beams?.length) {
    return [];
  }
  const grouped = new Map<string, WeaponRow>();
  for (const beam of ship.beams) {
    const key = `${beam.name}:${beam.maxDamage}`;
    const row = grouped.get(key);
    if (row) {
      row.count += 1;
    } else {
      grouped.set(key, {
        label: beam.name,
        count: 1,
        damage: beam.maxDamage,
      });
    }
  }
  return [...grouped.values()];
}

export function BattlePage() {
  const params = useParams();
  const navigate = useNavigate();
  const { state, getBattle, sendBattleAction, refreshGame, setInfo } =
    useAppContext();

  const [battle, setBattle] = createSignal<ApiTacticalBattle | null>(null);
  const [inspected, setInspected] = createSignal<ApiTacticalShip | null>(null);
  const [requestBusy, setRequestBusy] = createSignal(false);
  const [animating, setAnimating] = createSignal(true);
  const [autoMode, setAutoMode] = createSignal(false);
  const [error, setError] = createSignal("");

  let stageRef: HTMLDivElement | undefined;
  let phaserGame: Phaser.Game | null = null;
  let scene: BattleScene | null = null;
  let leaving = false;

  const playerId = () => state.session.playerId;
  const battleId = () => Number.parseInt(params.battleId ?? "", 10);

  const colorByOwner = createMemo(() => {
    const colors: Record<number, string> = {};
    for (const player of state.game?.players ?? []) {
      colors[player.id] = player.color;
    }
    return colors;
  });

  const activeShip = createMemo(() => {
    const current = battle();
    return current ? currentBattleShip(current) : null;
  });
  const isMyTurn = createMemo(
    () => activeShip()?.owner_id === playerId() && battle()?.status === "active",
  );
  const canAct = createMemo(
    () => isMyTurn() && !requestBusy() && !animating(),
  );
  // HUD focuses on the acting ship; hovering another ship inspects it instead.
  const hudShip = createMemo(() => inspected() ?? activeShip());

  const statusLine = createMemo(() => {
    const current = battle();
    if (!current) return "";
    if (current.status === "finished") {
      if (current.draw) return "Stalemate — fleets disengage";
      return current.winner_owner_id === playerId()
        ? "Victory!"
        : "Fleet lost";
    }
    const ship = activeShip();
    if (!ship) return `Round ${current.round}`;
    const side = ship.owner_id === playerId() ? "Your" : "Enemy";
    return `Round ${current.round} — ${side} ship: ${ship.name}`;
  });

  function leaveBattle() {
    if (leaving) return;
    leaving = true;
    void refreshGame()
      .catch(() => {
        // banner already set by refreshGame
      })
      .finally(() => navigate("/galaxy"));
  }

  function handleSettled(latest: ApiTacticalBattle) {
    setAnimating(false);
    setBattle({ ...latest });
    if (latest.status === "finished") {
      window.setTimeout(leaveBattle, RETURN_TO_MAP_DELAY_MS);
      return;
    }
    if (autoMode()) {
      void act({ type: "auto" });
    }
  }

  async function act(action: BattleActionRequest) {
    const current = battle();
    if (!current || current.status !== "active" || requestBusy()) return;
    const ship = currentBattleShip(current);
    if (!ship || ship.owner_id !== playerId()) return;
    setRequestBusy(true);
    setError("");
    try {
      const updated = await sendBattleAction(current.id, {
        ...action,
        ship_id: ship.id,
      });
      setAnimating(true);
      setBattle({ ...updated });
      if (!scene && phaserGame) {
        scene = phaserGame.scene.getScene("battle") as BattleScene;
      }
      scene?.applyBattle(updated);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Battle action failed.",
      );
    } finally {
      setRequestBusy(false);
    }
  }

  function handleIntent(intent: BattleIntent) {
    if (!canAct()) return;
    if (intent.type === "move") {
      void act({ type: "move", x: intent.x, y: intent.y });
    } else {
      void act({ type: "fire", target_id: intent.targetId });
    }
  }

  function toggleAuto() {
    const next = !autoMode();
    setAutoMode(next);
    if (next && canAct()) {
      void act({ type: "auto" });
    }
  }

  // Phaser game lifecycle — only create once the container has real dimensions.
  let gameStarted = false;
  function startGame(loaded: ApiTacticalBattle) {
    if (gameStarted || !stageRef) return;
    const w = stageRef.clientWidth;
    const h = stageRef.clientHeight;
    if (w < 10 || h < 10) return; // not laid out yet
    gameStarted = true;

    phaserGame = new Phaser.Game({
      type: Phaser.AUTO,
      parent: stageRef,
      backgroundColor: "#05060d",
      disableContextMenu: true,
      scale: {
        mode: Phaser.Scale.RESIZE,
        width: w,
        height: h,
      },
    });
    phaserGame.scene.add("battle", BattleScene, true, {
      battle: loaded,
      playerId: playerId(),
      colorByOwner: colorByOwner(),
      onIntent: handleIntent,
      onSettled: handleSettled,
      onInspectShip: (ship: ApiTacticalShip | null) => setInspected(ship),
    });
    scene = phaserGame.scene.getScene("battle") as BattleScene;
  }

  onMount(async () => {
    if (!state.session.selectedGameId) {
      navigate("/");
      return;
    }
    let loaded: ApiTacticalBattle;
    try {
      loaded = await getBattle(battleId());
    } catch {
      setInfo("Battle already resolved");
      leaveBattle();
      return;
    }
    setBattle(loaded);
    startGame(loaded);
  });

  // If the container wasn't ready during onMount, watch for it to become ready.
  createEffect(() => {
    const b = battle();
    if (!b || gameStarted || !stageRef) return;
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width, height } = entry.contentRect;
        if (width >= 10 && height >= 10) {
          ro.disconnect();
          startGame(b);
          return;
        }
      }
    });
    ro.observe(stageRef);
    onCleanup(() => ro.disconnect());
  });

  onCleanup(() => {
    gameStarted = false;
    phaserGame?.destroy(true);
    phaserGame = null;
    scene = null;
  });

  return (
    <div class="battle-page">
      <div class="battle-stage" ref={stageRef}>
        <Show when={battle()} fallback={<div class="battle-empty">ENTERING COMBAT…</div>}>
          <div class="battle-stage-status">{statusLine()}</div>
        </Show>
      </div>

      <div class="battle-hud">
        <section class="battle-hud-panel" aria-label="Ship">
          <h3 class="battle-hud-title">
            {hudShip()
              ? hudShip()!.owner_id === playerId()
                ? "Your Ship"
                : "Enemy Ship"
              : "Ship"}
          </h3>
          <Show when={hudShip()} fallback={<p class="battle-ship-sub">—</p>}>
            <p
              class="battle-ship-name"
              style={{
                color: colorByOwner()[hudShip()!.owner_id] ?? "var(--text)",
              }}
            >
              {hudShip()!.name}
            </p>
            <p class="battle-ship-sub">
              {hudShip()!.design_name || "Civilian hull"}
              <Show when={hudShip()!.hull_size}> · {hudShip()!.hull_size}</Show>
            </p>
          </Show>
        </section>

        <section class="battle-hud-panel" aria-label="Weapons">
          <h3 class="battle-hud-title">Weapons</h3>
          <Show
            when={weaponRows(hudShip()).length > 0}
            fallback={<p class="battle-ship-sub">No weapons</p>}
          >
            <For each={weaponRows(hudShip())}>
              {(row) => (
                <div
                  class="battle-weapon-row"
                  classList={{ "is-spent": hudShip()?.has_fired ?? false }}
                >
                  <span>
                    {row.count} × {row.label}
                  </span>
                  <span>dmg {row.damage}</span>
                </div>
              )}
            </For>
            <Show when={hudShip()?.has_fired}>
              <p class="battle-ship-sub">Fired this round</p>
            </Show>
          </Show>
        </section>

        <div class="battle-buttons" role="group" aria-label="Battle commands">
          <button
            type="button"
            class="secondary-button battle-button"
            classList={{ "is-toggled": autoMode() }}
            aria-pressed={autoMode()}
            onClick={toggleAuto}
            disabled={battle()?.status !== "active"}
          >
            Auto
          </button>
          <button type="button" class="secondary-button battle-button" disabled title="Coming soon">
            Scan
          </button>
          <button type="button" class="secondary-button battle-button" disabled title="Coming soon">
            Board
          </button>
          <button type="button" class="secondary-button battle-button" disabled title="Coming soon">
            Retreat
          </button>
          <button
            type="button"
            class="secondary-button battle-button"
            onClick={() => void act({ type: "wait" })}
            disabled={!canAct()}
          >
            Wait
          </button>
          <button
            type="button"
            class="secondary-button battle-button"
            onClick={() => void act({ type: "done" })}
            disabled={!canAct()}
          >
            Done
          </button>
        </div>

        <section class="battle-hud-panel" aria-label="Systems">
          <h3 class="battle-hud-title">Systems</h3>
          <Show when={hudShip()} fallback={<p class="battle-ship-sub">—</p>}>
            <dl class="battle-systems-grid">
              <dt>Structure</dt>
              <dd>
                {hudShip()!.status.current_structure_hp}/
                {hudShip()!.status.max_structure_hp}
              </dd>
              <dt>Armor</dt>
              <dd>
                {hudShip()!.status.current_armor_hp}/
                {hudShip()!.status.max_armor_hp}
              </dd>
              <dt>Computer</dt>
              <dd>+{hudShip()!.beam_attack_bonus}</dd>
              <dt>Defense</dt>
              <dd>{hudShip()!.beam_defense}</dd>
              <dt>Max Speed</dt>
              <dd>{hudShip()!.combat_speed}</dd>
              <dt>Remaining</dt>
              <dd>{hudShip()!.moves_left}</dd>
            </dl>
          </Show>
        </section>
      </div>

      <Show when={error()}>
        <div class="status-banner error" role="alert">
          {error()}
        </div>
      </Show>
    </div>
  );
}
