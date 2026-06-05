import {
  For,
  createEffect,
  createMemo,
  createSignal,
  onCleanup,
  onMount,
} from "solid-js";

import type { StarSummary } from "../lib/types/domain";
import {
  GALAXY_VIEWPORT_STORAGE_PREFIX,
  ZOOM_STEP,
  centerViewportOn,
  clampViewport,
  getInitialGalaxyViewport,
  getZoom,
  isAtMaxZoom,
  normalizeGalaxyMapBounds,
  panViewportByScreenDelta,
  parseStoredGalaxyViewport,
  screenToSvgPoint,
  viewportToViewBox,
  zoomViewportAt,
  type GalaxyMapViewport,
} from "../lib/galaxy-viewport";
import { STAR_COLORS } from "../lib/utils";

import "./GalaxyMap.css";

const DRAG_CLICK_SUPPRESSION_PX = 4;
const KEYBOARD_PAN_UNITS = 40;
const STORAGE_DEBOUNCE_MS = 150;
const STAR_VIEW_MARGIN = 28;

type GalaxyMapProps = {
  stars: StarSummary[];
  selectedStarId?: number;
  onSelect: (starId: number) => void;
  fleets?: {
    id: number;
    starId: number;
    x: number;
    y: number;
    isMoving?: boolean;
  }[];
  onFleetClick?: (fleetId: number) => void;
  routeLines?: {
    fromX: number;
    fromY: number;
    toX: number;
    toY: number;
    eta: number;
    preview?: boolean;
    blocked?: boolean;
    starGate?: boolean;
  }[];
  onHoverSnappedStar?: (starId: number | null) => void;
  gameId?: string;
  width?: number;
  height?: number;
};

export function GalaxyMap(props: GalaxyMapProps) {
  let svgRef: SVGSVGElement | undefined;
  let saveTimeout: ReturnType<typeof setTimeout> | undefined;
  const starsContentBounds = createMemo(() => {
    if (props.stars.length === 0) {
      return undefined;
    }

    const xs = props.stars.map((star) => star.x);
    const ys = props.stars.map((star) => star.y);
    return {
      minX: Math.min(...xs) - STAR_VIEW_MARGIN,
      minY: Math.min(...ys) - STAR_VIEW_MARGIN,
      maxX: Math.max(...xs) + STAR_VIEW_MARGIN,
      maxY: Math.max(...ys) + STAR_VIEW_MARGIN,
    };
  });
  const mapBounds = createMemo(() =>
    normalizeGalaxyMapBounds(props.width, props.height, starsContentBounds()),
  );
  const initialViewport = createMemo(() =>
    getInitialGalaxyViewport(mapBounds()),
  );
  const [viewport, setViewport] =
    createSignal<GalaxyMapViewport>(initialViewport());
  const [isDragging, setIsDragging] = createSignal(false);
  const [dragState, setDragState] = createSignal<{
    pointerId: number;
    lastX: number;
    lastY: number;
    totalMovement: number;
  } | null>(null);
  const [suppressNextClick, setSuppressNextClick] = createSignal(false);
  const viewBox = createMemo(() => viewportToViewBox(viewport()));
  const zoom = createMemo(() => getZoom(viewport(), mapBounds()));
  const canZoomIn = createMemo(() => !isAtMaxZoom(viewport(), mapBounds()));

  function storageKey() {
    const bounds = mapBounds();
    return props.gameId
      ? `${GALAXY_VIEWPORT_STORAGE_PREFIX}${props.gameId}.${bounds.width}x${bounds.height}`
      : "";
  }

  function loadStoredViewport() {
    const key = storageKey();
    if (!key) {
      setViewport(initialViewport());
      return;
    }
    try {
      setViewport(
        parseStoredGalaxyViewport(
          window.localStorage.getItem(key),
          mapBounds(),
        ),
      );
    } catch {
      setViewport(initialViewport());
    }
  }

  onMount(loadStoredViewport);

  createEffect(() => {
    props.gameId;
    props.width;
    props.height;
    loadStoredViewport();
  });

  createEffect(() => {
    const key = storageKey();
    const nextViewport = viewport();
    if (!key) {
      return;
    }
    if (saveTimeout) {
      clearTimeout(saveTimeout);
    }
    saveTimeout = setTimeout(() => {
      try {
        window.localStorage.setItem(key, JSON.stringify(nextViewport));
      } catch {
        /* quota / private mode */
      }
    }, STORAGE_DEBOUNCE_MS);
  });

  onCleanup(() => {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
    }
  });

  function svgBounds() {
    return svgRef?.getBoundingClientRect() ?? null;
  }

  function svgPointForClient(clientX: number, clientY: number) {
    const bounds = svgBounds();
    if (!bounds) {
      return null;
    }
    return screenToSvgPoint({ clientX, clientY }, bounds, viewport());
  }

  function zoomAt(clientX: number, clientY: number, scale: number) {
    if (scale > 1 && !canZoomIn()) {
      return;
    }
    const svgPoint = svgPointForClient(clientX, clientY);
    if (!svgPoint) {
      return;
    }
    setViewport((current) =>
      zoomViewportAt(current, svgPoint, scale, mapBounds()),
    );
  }

  function zoomAtCenter(scale: number) {
    if (scale > 1 && !canZoomIn()) {
      return;
    }
    const current = viewport();
    setViewport(
      zoomViewportAt(
        current,
        {
          x: current.x + current.width / 2,
          y: current.y + current.height / 2,
        },
        scale,
        mapBounds(),
      ),
    );
  }

  function panByScreenDelta(deltaClientX: number, deltaClientY: number) {
    const bounds = svgBounds();
    if (!bounds) {
      return;
    }
    setViewport((current) =>
      panViewportByScreenDelta(
        current,
        mapBounds(),
        bounds,
        deltaClientX,
        deltaClientY,
      ),
    );
  }

  function resetViewport() {
    setViewport(initialViewport());
  }

  function focusSelectedStar() {
    const selectedStar = props.stars.find(
      (star) => star.id === props.selectedStarId,
    );
    if (!selectedStar) {
      return;
    }
    setViewport((current) =>
      centerViewportOn(current, selectedStar, mapBounds()),
    );
  }

  function handleMouseMove(event: MouseEvent) {
    if (!props.onHoverSnappedStar) {
      return;
    }
    const point = svgPointForClient(event.clientX, event.clientY);
    if (!point) {
      return;
    }
    const snapDistance = 42;
    let nearest: { id: number; distance: number } | null = null;
    for (const star of props.stars) {
      const dx = star.x - point.x;
      const dy = star.y - point.y;
      const distance = Math.sqrt(dx * dx + dy * dy);
      if (distance > snapDistance) {
        continue;
      }
      if (!nearest || distance < nearest.distance) {
        nearest = { id: star.id, distance };
      }
    }
    props.onHoverSnappedStar(nearest?.id ?? null);
  }

  function handleWheel(event: WheelEvent) {
    event.preventDefault();
    zoomAt(
      event.clientX,
      event.clientY,
      event.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP,
    );
  }

  function handlePointerDown(event: PointerEvent) {
    if (!event.isPrimary || event.button !== 0) {
      return;
    }
    const target = event.target as Element;
    if (target.closest(".galaxy-star, .star-fleet-link, .galaxy-map-control")) {
      return;
    }
    event.preventDefault();
    svgRef?.setPointerCapture(event.pointerId);
    setDragState({
      pointerId: event.pointerId,
      lastX: event.clientX,
      lastY: event.clientY,
      totalMovement: 0,
    });
    setIsDragging(true);
  }

  function handlePointerMove(event: PointerEvent) {
    handleMouseMove(event);
    const drag = dragState();
    if (!drag || drag.pointerId !== event.pointerId) {
      return;
    }
    const deltaX = event.clientX - drag.lastX;
    const deltaY = event.clientY - drag.lastY;
    const movement = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
    panByScreenDelta(deltaX, deltaY);
    setDragState({
      pointerId: drag.pointerId,
      lastX: event.clientX,
      lastY: event.clientY,
      totalMovement: drag.totalMovement + movement,
    });
  }

  function stopDragging(event: PointerEvent) {
    const drag = dragState();
    if (
      drag?.pointerId === event.pointerId &&
      drag.totalMovement > DRAG_CLICK_SUPPRESSION_PX
    ) {
      setSuppressNextClick(true);
    }
    setDragState(null);
    setIsDragging(false);
  }

  function handleStarClick(starId: number) {
    if (suppressNextClick()) {
      setSuppressNextClick(false);
      return;
    }
    props.onSelect(starId);
  }

  function handleKeyDown(event: KeyboardEvent) {
    const current = viewport();
    const panDistance = KEYBOARD_PAN_UNITS / zoom();
    switch (event.key) {
      case "ArrowUp":
        event.preventDefault();
        setViewport(
          clampViewport(
            { ...current, y: current.y - panDistance },
            mapBounds(),
          ),
        );
        break;
      case "ArrowDown":
        event.preventDefault();
        setViewport(
          clampViewport(
            { ...current, y: current.y + panDistance },
            mapBounds(),
          ),
        );
        break;
      case "ArrowLeft":
        event.preventDefault();
        setViewport(
          clampViewport(
            { ...current, x: current.x - panDistance },
            mapBounds(),
          ),
        );
        break;
      case "ArrowRight":
        event.preventDefault();
        setViewport(
          clampViewport(
            { ...current, x: current.x + panDistance },
            mapBounds(),
          ),
        );
        break;
      case "+":
      case "=":
        event.preventDefault();
        zoomAtCenter(ZOOM_STEP);
        break;
      case "-":
        event.preventDefault();
        zoomAtCenter(1 / ZOOM_STEP);
        break;
      case "0":
        event.preventDefault();
        resetViewport();
        break;
      case "f":
      case "F":
        if (props.selectedStarId) {
          event.preventDefault();
          focusSelectedStar();
        }
        break;
    }
  }

  return (
    <div class="galaxy-map-shell">
      <div class="galaxy-map-toolbar" aria-label="Galaxy map controls">
        <button
          class="galaxy-map-control"
          type="button"
          aria-label="Zoom in"
          title="Zoom in"
          disabled={!canZoomIn()}
          onClick={() => zoomAtCenter(ZOOM_STEP)}
        >
          +
        </button>
        <button
          class="galaxy-map-control"
          type="button"
          aria-label="Zoom out"
          title="Zoom out"
          onClick={() => zoomAtCenter(1 / ZOOM_STEP)}
        >
          -
        </button>
        <button
          class="galaxy-map-control"
          type="button"
          aria-label="Reset view"
          title="Reset view"
          onClick={resetViewport}
        >
          0
        </button>
        <button
          class="galaxy-map-control"
          type="button"
          aria-label="Focus selected star"
          title="Focus selected star"
          disabled={!props.selectedStarId}
          onClick={focusSelectedStar}
        >
          ⌖
        </button>
        <span class="galaxy-map-zoom-readout">{Math.round(zoom() * 100)}%</span>
      </div>
      <div class="galaxy-map-frame">
        <svg
          ref={svgRef}
          class="galaxy-map"
          classList={{ "is-dragging": isDragging() }}
          viewBox={viewBox()}
          role="application"
          aria-label="Galaxy map. Use arrow keys to pan, plus and minus to zoom, and zero to reset."
          tabIndex={0}
          onWheel={handleWheel}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={stopDragging}
          onPointerCancel={stopDragging}
          onLostPointerCapture={() => {
            setDragState(null);
            setIsDragging(false);
          }}
          onKeyDown={handleKeyDown}
          onMouseLeave={() => props.onHoverSnappedStar?.(null)}
        >
          <g id="galaxy-stars">
            <For each={props.stars}>
              {(star) => (
                <g
                  class="galaxy-star"
                  classList={{ selected: props.selectedStarId === star.id }}
                  transform={`translate(${star.x} ${star.y})`}
                  onClick={() => handleStarClick(star.id)}
                >
                  <circle
                    class="star-halo"
                    r="10"
                    fill={STAR_COLORS[star.color] ?? STAR_COLORS[0]}
                  />
                  <circle
                    class="star-core"
                    r="4.5"
                    fill={STAR_COLORS[star.color] ?? STAR_COLORS[0]}
                  />
                  <circle class="star-ring" r="12" />
                  <text
                    x="0"
                    y="22"
                    text-anchor="middle"
                    classList={{
                      "star-label-unoccupied":
                        star.explored && star.ownedPlanets === 0,
                    }}
                    style={
                      star.ownerColor ? { fill: star.ownerColor } : undefined
                    }
                  >
                    {star.name}
                  </text>
                </g>
              )}
            </For>
          </g>
          <g id="fleet-links">
            <For each={props.fleets ?? []}>
              {(fleet) => (
                <g
                  class="star-fleet-link"
                  transform={`translate(${fleet.x + (fleet.isMoving ? 0 : 14)} ${fleet.y + (fleet.isMoving ? 0 : -10)})`}
                  onClick={(event) => {
                    event.stopPropagation();
                    props.onFleetClick?.(fleet.id);
                  }}
                >
                  <circle r="7.5" />
                  <text text-anchor="middle" y="3.5">
                    🚀
                  </text>
                </g>
              )}
            </For>
          </g>
          <g class="fleet-route-layer">
            <For each={props.routeLines ?? []}>
              {(route) => {
                const midX = (route.fromX + route.toX) / 2;
                const midY = (route.fromY + route.toY) / 2;
                return (
                  <g classList={{ "is-preview": Boolean(route.preview), "is-blocked": Boolean(route.blocked), "is-star-gate": Boolean(route.starGate) }}>
                    <line
                      x1={route.fromX}
                      y1={route.fromY}
                      x2={route.toX}
                      y2={route.toY}
                      class="fleet-route-line"
                      classList={{ "star-gate-line": Boolean(route.starGate) }}
                    />
                    <text
                      x={midX}
                      y={midY - 8}
                      text-anchor="middle"
                      class="fleet-route-eta"
                    >
                      {route.starGate ? "STAR GATE" : `ETA ${route.eta}`}
                    </text>
                  </g>
                );
              }}
            </For>
          </g>
        </svg>
      </div>
    </div>
  );
}
