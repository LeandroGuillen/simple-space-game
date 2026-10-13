import { A, useNavigate, useParams } from "@solidjs/router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { Portal } from "solid-js/web";
import { attachModalBehavior } from "../lib/modal";
import { ConstructionPanel } from "../components/ConstructionPanel";
import { FleetOrbitModal } from "../components/FleetOrbitModal";
import { PlanetVisual } from "../components/PlanetVisual";
import { PopulationPanel } from "../components/PopulationPanel";
import { planetReturnPath } from "../lib/navigation-history";
import { createShortcut } from "../lib/shortcuts";
import {
  findPlanet,
  findPlayerFleetsAtStar,
  findStar,
  calculateFleetTravelTurnsFromCoordinates,
  getPlanetProjectedIncome,
  getPlanetProjectedPopulationGrowth,
  labelForEnvironment,
  labelForGravity,
  labelForMinerals,
  labelForSize,
  sortFleetsForStarTabs,
} from "../lib/utils";
import { useAppContext } from "../state/app-context";

export function PlanetPage() {
  const params = useParams();
  const navigate = useNavigate();
  const { state, currentPlayer, addTransferPopulationCommand } = useAppContext();
  const [isFleetModalOpen, setIsFleetModalOpen] = createSignal(false);
  const [isTransferModalOpen, setIsTransferModalOpen] = createSignal(false);
  const [transferDestinationId, setTransferDestinationId] = createSignal<number | null>(null);
  const [transferUnits, setTransferUnits] = createSignal(1);

  createShortcut("Escape", () => {
    navigate(planetReturnPath());
  });
  const planet = () => findPlanet(state.game, Number(params.planetId));
  const populationUnits = (population: number) => Math.floor(population / 1000);
  const hasPopulationCapacity = (maxPopulation: number) =>
    populationUnits(maxPopulation) > 0;
  const playerColorById = createMemo(
    () =>
      new Map(
        (state.game?.players ?? []).map((player) => [player.id, player.color]),
      ),
  );
  const planetIconSize = (size: number) =>
    `${[14, 16, 21, 26, 31][size - 1] ?? 22}px`;
  const planetOwnerColor = (owner: number) =>
    playerColorById().get(owner) ?? "var(--text-mute)";
  const canVisitPlanet = (owner: number) => owner === currentPlayer()?.id;
  const star = createMemo(() => {
    const current = planet();
    return current ? findStar(state.game, current.parentStar) : null;
  });
  const projectedPopGrowth = createMemo(() => {
    const current = planet();
    return current
      ? getPlanetProjectedPopulationGrowth(
          current,
          state.pendingCommands,
          currentPlayer()?.raceTraitIds,
          currentPlayer()?.racePopulationGrowthPercent,
        )
      : 0;
  });
  const incomeBreakdown = createMemo(() => {
    const current = planet();
    if (!current) {
      return {
        taxes: 0,
        buildings: 0,
        tradeGoods: 0,
        maintenance: 0,
        total: 0,
        bonus: 0,
      };
    }

    return getPlanetProjectedIncome(current, state.pendingCommands, currentPlayer()?.raceTraitIds);
  });
  const incomeIcons = createMemo(() => {
    const total = Math.max(0, incomeBreakdown().total);
    const bags = Math.floor(total / 10);
    const coins = total % 10;
    return [
      ...Array.from({ length: bags }, () => "💰"),
      ...Array.from({ length: coins }, () => "🪙"),
    ];
  });
  const gravityEffect = createMemo(() => {
    const current = planet();
    if (!current) return null;
    const multiplier = current.gravityMultiplier ?? 1;
    const traits = currentPlayer()?.raceTraitIds ?? [];
    const adapted = traits.includes("low_g_world") || traits.includes("high_g_world");
    if (multiplier === 1) {
      return adapted ? { label: "Adapted", tone: "good" as const } : null;
    }
    const percent = Math.round((multiplier - 1) * 100);
    return { label: `${percent}% farming/production/research`, tone: "bad" as const };
  });
  const formatMoralePercent = (bonus: number) =>
    `${bonus >= 0 ? "+" : ""}${Math.round(bonus * 100)}%`;
  const formatMultiplierPercent = (multiplier: number) => {
    const percent = Math.round((multiplier - 1) * 100);
    return `${percent >= 0 ? "+" : ""}${percent}%`;
  };
  const moraleIcons = createMemo(() => {
    const bonus = planet()?.morale.bonus ?? 0;
    return Array.from({
      length: Math.max(0, Math.floor(bonus * 10 + 0.000001)),
    });
  });
  const fleetsAtParentStar = createMemo(() => {
    const current = planet();
    const playerId = currentPlayer()?.id ?? 0;
    if (!current || !playerId) {
      return [];
    }
    return sortFleetsForStarTabs(
      findPlayerFleetsAtStar(state.game, playerId, current.parentStar),
    );
  });
  const fleetShipCountAtParentStar = createMemo(() =>
    fleetsAtParentStar().reduce((total, f) => total + f.ships.length, 0),
  );
  const ownedColonies = createMemo(() => {
    const playerId = currentPlayer()?.id ?? 0;
    return (state.game?.galaxy.stars ?? [])
      .flatMap((system) =>
        system.planets
          .filter((candidate) => candidate.owner === playerId)
          .map((candidate) => ({ planet: candidate, star: system })),
      )
      .sort((a, b) => a.star.name.localeCompare(b.star.name) || a.planet.name.localeCompare(b.planet.name));
  });
  const transferDestinations = createMemo(() => {
    const current = planet();
    return ownedColonies().filter((item) => item.planet.id !== current?.id);
  });
  const selectedTransferDestination = createMemo(() => {
    const destinations = transferDestinations();
    if (destinations.length === 0) {
      return null;
    }
    const selectedId = transferDestinationId();
    return destinations.find((item) => item.planet.id === selectedId) ?? destinations[0] ?? null;
  });
  const availableTransferFreighters = createMemo(() => {
    const player = currentPlayer();
    if (!player) return 0;
    const pendingReserved = (state.pendingCommands.transferPopulation ?? []).reduce(
      (total, command) => total + command.population_units * 5,
      0,
    );
    return Math.max(0, player.freighters - player.freightersUsed - pendingReserved);
  });
  const maxTransferUnits = createMemo(() => {
    const current = planet();
    const destination = selectedTransferDestination()?.planet;
    if (!current || !destination) return 0;
    const pendingOut = (state.pendingCommands.transferPopulation ?? [])
      .filter((command) => command.source_planet_id === current.id)
      .reduce((total, command) => total + command.population_units, 0);
    const pendingIn = (state.pendingCommands.transferPopulation ?? [])
      .filter((command) => command.destination_planet_id === destination.id)
      .reduce((total, command) => total + command.population_units, 0);
    const sourceUnits = Math.floor(current.population / 1000);
    const destinationFree = Math.floor(destination.maxPopulation / 1000) - Math.floor(destination.population / 1000) - pendingIn;
    return Math.max(
      0,
      Math.min(sourceUnits - 1 - pendingOut, destinationFree, Math.floor(availableTransferFreighters() / 5)),
    );
  });
  const transferEta = createMemo(() => {
    const current = planet();
    const destination = selectedTransferDestination();
    const originStar = current ? findStar(state.game, current.parentStar) : null;
    if (!originStar || !destination) return 0;
    return calculateFleetTravelTurnsFromCoordinates(
      originStar.x,
      originStar.y,
      destination.star.x,
      destination.star.y,
    );
  });
  const requestedTransferUnits = createMemo(() => {
    const units = transferUnits();
    return Number.isInteger(units) && units > 0
      ? Math.min(units, Math.max(1, maxTransferUnits()))
      : 0;
  });
  const selectedTransferUnits = createMemo(() =>
    Math.min(requestedTransferUnits(), maxTransferUnits()),
  );
  const transferBlockedReason = createMemo(() => {
    if (availableTransferFreighters() < 5) return "You need 5 available freighters to transfer one population unit.";
    if (maxTransferUnits() <= 0) return "There is no population available to send or no room at this destination, including queued transfers.";
    return null;
  });
  const closeTransferModal = () => setIsTransferModalOpen(false);
  const openTransferModal = () => {
    setTransferDestinationId(null);
    setTransferUnits(1);
    setIsTransferModalOpen(true);
  };

  function queuePopulationTransfer() {
    const current = planet();
    const destination = selectedTransferDestination();
    const units = selectedTransferUnits();
    if (!current || !destination || units <= 0 || state.isBusy) return;
    addTransferPopulationCommand({
      source_planet_id: current.id,
      destination_planet_id: destination.planet.id,
      population_units: units,
    });
    closeTransferModal();
  }

  return (
    <Show
      when={planet()}
      fallback={<section class="panel">Planet not found.</section>}
    >
      {(currentPlanet) => (
        <section class="page-grid">
          <section class="panel">
            <div class="feature-header">
              <Show when={star()?.planets.length}>
                <nav
                  class="system-planet-nav"
                  aria-label={`${star()?.name ?? "System"} planets`}
                >
                  <For each={star()?.planets ?? []}>
                    {(systemPlanet) => (
                      <Show
                        when={canVisitPlanet(systemPlanet.owner)}
                        fallback={
                          <span
                            class="system-planet-link system-planet-link--disabled"
                            title={`${systemPlanet.name}: ${labelForMinerals(systemPlanet.mineralClass)}, ${populationUnits(systemPlanet.population)}/${populationUnits(systemPlanet.maxPopulation)} population`}
                          >
                            <span
                              class="system-planet-icon"
                              style={{
                                "--planet-nav-icon-size": planetIconSize(
                                  systemPlanet.size,
                                ),
                              }}
                            >
                              <PlanetVisual
                                planetId={systemPlanet.id}
                                environment={systemPlanet.environment}
                                size="small"
                              />
                            </span>
                            <span class="system-planet-meta">
                              <strong
                                style={{
                                  color:
                                    systemPlanet.owner > 0
                                      ? planetOwnerColor(systemPlanet.owner)
                                      : "var(--text-mute)",
                                }}
                              >
                                {systemPlanet.name}
                              </strong>
                              <span>
                                {labelForMinerals(systemPlanet.mineralClass)}
                              </span>
                              <Show
                                when={hasPopulationCapacity(
                                  systemPlanet.maxPopulation,
                                )}
                              >
                                <span>
                                  {populationUnits(systemPlanet.population)}/
                                  {populationUnits(systemPlanet.maxPopulation)}
                                </span>
                              </Show>
                            </span>
                          </span>
                        }
                      >
                        <A
                          class="system-planet-link"
                          classList={{
                            "system-planet-link--active":
                              systemPlanet.id === currentPlanet().id,
                          }}
                          href={`/stars/${systemPlanet.parentStar}/planets/${systemPlanet.id}`}
                          aria-current={
                            systemPlanet.id === currentPlanet().id
                              ? "page"
                              : undefined
                          }
                          title={`${systemPlanet.name}: ${labelForMinerals(systemPlanet.mineralClass)}, ${populationUnits(systemPlanet.population)}/${populationUnits(systemPlanet.maxPopulation)} population`}
                        >
                          <span
                            class="system-planet-icon"
                            style={{
                              "--planet-nav-icon-size": planetIconSize(
                                systemPlanet.size,
                              ),
                            }}
                          >
                            <PlanetVisual
                              planetId={systemPlanet.id}
                              environment={systemPlanet.environment}
                              size="small"
                            />
                          </span>
                          <span class="system-planet-meta">
                            <strong
                              style={{
                                color: planetOwnerColor(systemPlanet.owner),
                              }}
                            >
                              {systemPlanet.name}
                            </strong>
                            <span>
                              {labelForMinerals(systemPlanet.mineralClass)}
                            </span>
                            <Show
                              when={hasPopulationCapacity(
                                systemPlanet.maxPopulation,
                              )}
                            >
                              <span>
                                {populationUnits(systemPlanet.population)}/
                                {populationUnits(systemPlanet.maxPopulation)}
                              </span>
                            </Show>
                          </span>
                        </A>
                      </Show>
                    )}
                  </For>
                </nav>
              </Show>
              <PlanetVisual
                planetId={currentPlanet().id}
                environment={currentPlanet().environment}
                size="large"
              />
              <div class="planet-details">
                <h2>{currentPlanet().name}</h2>
                <p class="lede">
                  {labelForEnvironment(currentPlanet().environment)} ·{" "}
                  {labelForSize(currentPlanet().size)} ·{" "}
                  {labelForGravity(currentPlanet().gravity)} gravity ·{" "}
                  {labelForMinerals(currentPlanet().mineralClass)}
                  <Show when={gravityEffect()}>
                    {(effect) => (
                      <span
                        style={{
                          "margin-left": "0.5em",
                          color: effect().tone === "good" ? "var(--text-success, #4ade80)" : "var(--text-warning, #f87171)",
                        }}
                        title={`Gravity effect: ${effect().label}`}
                      >
                        ({effect().label})
                      </span>
                    )}
                  </Show>
                </p>
                <article class="metric-card">
                  <span>Population</span>
                  <strong>
                    {currentPlanet().population}/{currentPlanet().maxPopulation}
                  </strong>
                  <Show when={projectedPopGrowth() >= 0}>
                    <span>+{projectedPopGrowth()} this turn</span>
                  </Show>
                  <Show when={projectedPopGrowth() < 0}>
                    <span class="starving-warning">
                      {projectedPopGrowth()} this turn
                    </span>
                  </Show>
                </article>
                <article
                  class="metric-card income-metric"
                  classList={{
                    "income-metric--negative": incomeBreakdown().total < 0,
                  }}
                  aria-label={`Income ${incomeBreakdown().total} BC`}
                >
                  <span>Income</span>
                  <div class="income-value">
                    <Show
                      when={incomeBreakdown().total < 0}
                      fallback={
                        <For each={incomeIcons()}>
                          {(icon) => (
                            <span class="pop-stat-icon income-icon">
                              {icon}
                            </span>
                          )}
                        </For>
                      }
                    >
                      <strong>{incomeBreakdown().total} BC</strong>
                    </Show>
                  </div>
                  <div class="pop-output-tooltip income-tooltip">
                    <div class="pop-output-tooltip-row pop-output-tooltip-row--total">
                      <span>Total income:</span>
                      <strong>{incomeBreakdown().total} BC</strong>
                    </div>
                    <div class="pop-output-tooltip-row">
                      <span>Taxes:</span>
                      <strong>{incomeBreakdown().taxes} BC</strong>
                    </div>
                    <Show
                      when={
                        (currentPlanet().government?.taxMultiplier ?? 1) !== 1
                      }
                    >
                      <div class="pop-output-tooltip-row">
                        <span>
                          {currentPlanet().government?.name
                            ? `${currentPlanet().government?.name} bonus:`
                            : "Government bonus:"}
                        </span>
                        <strong>
                          {formatMultiplierPercent(
                            currentPlanet().government?.taxMultiplier ?? 1,
                          )}
                        </strong>
                      </div>
                    </Show>
                    <div class="pop-output-tooltip-row">
                      <span>Buildings:</span>
                      <strong>{incomeBreakdown().buildings} BC</strong>
                    </div>
                    <div class="pop-output-tooltip-row">
                      <span>Trade Goods:</span>
                      <strong>{incomeBreakdown().tradeGoods} BC</strong>
                    </div>
                    <Show when={incomeBreakdown().bonus !== 0}>
                      <div class="pop-output-tooltip-row">
                        <span>Income bonus:</span>
                        <strong>
                          +{Math.round(incomeBreakdown().bonus * 100)}%
                        </strong>
                      </div>
                    </Show>
                    <div class="pop-output-tooltip-row">
                      <span>Maintenance:</span>
                      <strong>-{incomeBreakdown().maintenance} BC</strong>
                    </div>
                  </div>
                </article>
                <article class="metric-card morale-metric">
                  <span>MORALE</span>
                  <div class="morale-icons">
                    <For each={moraleIcons()}>
                      {() => <span class="morale-icon">:)</span>}
                    </For>
                  </div>
                  <Show when={currentPlanet().morale.sources.length > 0}>
                    <div class="pop-output-tooltip morale-tooltip">
                      <div class="pop-output-tooltip-row pop-output-tooltip-row--total">
                        <span>Total morale:</span>
                        <strong>
                          {formatMoralePercent(currentPlanet().morale.bonus)}
                        </strong>
                      </div>
                      <For each={currentPlanet().morale.sources}>
                        {(source) => (
                          <div class="pop-output-tooltip-row">
                            <span>{source.name}</span>
                            <strong>{formatMoralePercent(source.bonus)}</strong>
                          </div>
                        )}
                      </For>
                    </div>
                  </Show>
                </article>
                <Show when={fleetsAtParentStar().length > 0}>
                  <button
                    class="fleet-orbit-link-inline fleet-orbit-link"
                    type="button"
                    onClick={() => setIsFleetModalOpen(true)}
                  >
                    🚀 Fleets ({fleetShipCountAtParentStar()})
                  </button>
                </Show>
              </div>
              <ConstructionPanel planet={currentPlanet()} />
            </div>
          </section>

          <PopulationPanel
            planet={currentPlanet()}
            onTransfer={currentPlanet().owner === currentPlayer()?.id ? openTransferModal : undefined}
          />
          <Show when={isTransferModalOpen()}>
            <Portal>
              <div class="research-modal-backdrop" onClick={closeTransferModal}>
                <section
                  class="research-modal panel transfer-pop-modal"
                  role="dialog"
                  aria-modal="true"
                  aria-labelledby="transfer-pop-title"
                  ref={(element) => attachModalBehavior(element, closeTransferModal)}
                  onClick={(event) => event.stopPropagation()}
                >
                  <header class="research-modal-header">
                    <div>
                      <p class="eyebrow">{currentPlanet().name}</p>
                      <h2 id="transfer-pop-title">Transfer population</h2>
                    </div>
                    <button class="link-button" type="button" onClick={closeTransferModal}>
                      Close
                    </button>
                  </header>
                  <Show
                    when={transferDestinations().length > 0}
                    fallback={<p class="lede">Colonize another planet before transferring population.</p>}
                  >
                    <form class="stack-form" onSubmit={(event) => {
                      event.preventDefault();
                      queuePopulationTransfer();
                    }}>
                      <label class="field">
                        <span>Destination colony</span>
                        <select
                          value={selectedTransferDestination()?.planet.id ?? ""}
                          onChange={(event) => {
                            setTransferDestinationId(Number(event.currentTarget.value));
                            setTransferUnits(1);
                          }}
                        >
                          <For each={transferDestinations()}>
                            {(item) => (
                              <option value={item.planet.id}>
                                {item.star.name} · {item.planet.name}
                              </option>
                            )}
                          </For>
                        </select>
                      </label>
                      <label class="field">
                        <span>Population units</span>
                        <input
                          type="number"
                          min="1"
                          max={Math.max(1, maxTransferUnits())}
                          step="1"
                          required
                          disabled={maxTransferUnits() <= 0}
                          value={Number.isFinite(transferUnits()) ? Math.min(transferUnits(), Math.max(1, maxTransferUnits())) : ""}
                          onInput={(event) => setTransferUnits(event.currentTarget.valueAsNumber)}
                          aria-describedby="transfer-pop-help"
                        />
                      </label>
                      <p class="transfer-pop-help" id="transfer-pop-help">
                        Each unit is 1,000 colonists and requires 5 freighters. Transfers depart next turn.
                      </p>
                      <div class="transfer-pop-summary">
                        <article class="stat-card">
                          <span>Travel time</span>
                          <strong>{transferEta()} turn{transferEta() === 1 ? "" : "s"}</strong>
                        </article>
                        <article class="stat-card">
                          <span>Freighters needed / available</span>
                          <strong>{requestedTransferUnits() * 5} / {availableTransferFreighters()}</strong>
                        </article>
                      </div>
                      <Show when={transferBlockedReason()}>
                        <p class="transfer-pop-notice" role="status">{transferBlockedReason()}</p>
                      </Show>
                      <footer class="form-actions">
                        <button class="secondary-button" type="button" onClick={closeTransferModal}>
                          Cancel
                        </button>
                        <button
                          class="primary-button"
                          type="submit"
                          disabled={selectedTransferUnits() <= 0 || state.isBusy}
                        >
                          Queue transfer
                        </button>
                      </footer>
                    </form>
                  </Show>
                </section>
              </div>
            </Portal>
          </Show>
          <FleetOrbitModal
            starId={currentPlanet().parentStar}
            isOpen={isFleetModalOpen()}
            onClose={() => setIsFleetModalOpen(false)}
          />
        </section>
      )}
    </Show>
  );
}
