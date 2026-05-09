import { For } from "solid-js";

import type { StarSummary } from "../lib/types/domain";
import { STAR_COLORS } from "../lib/utils";

import "./GalaxyMap.css";

/** Must match server galaxy bounds (`server/game/galaxy_utils.go`). */
const GALAXY_W = 700;
const GALAXY_H = 400;
/** One parsec maps to this many SVG user units (screen pixels when the map is shown 1:1). */
const MAP_PIX_W = GALAXY_W;
const MAP_PIX_H = GALAXY_H;
/** Margin in SVG units so halos, selection ring, and names do not clip (independent of parsec scale). */
const MAP_VIEW_PADDING = 100;

type GalaxyMapProps = {
  stars: StarSummary[];
  selectedStarId?: number;
  onSelect: (starId: number) => void;
  fleets?: { id: number; starId: number; x: number; y: number; isMoving?: boolean }[];
  onFleetClick?: (fleetId: number) => void;
  routeLines?: { fromX: number; fromY: number; toX: number; toY: number; eta: number; preview?: boolean }[];
  onHoverSnappedStar?: (starId: number | null) => void;
};

export function GalaxyMap(props: GalaxyMapProps) {
  const vbW = MAP_PIX_W + 2 * MAP_VIEW_PADDING;
  const vbH = MAP_PIX_H + 2 * MAP_VIEW_PADDING;
  const viewBox = `${-MAP_VIEW_PADDING} ${-MAP_VIEW_PADDING} ${vbW} ${vbH}`;
  function handleMouseMove(event: MouseEvent) {
    if (!props.onHoverSnappedStar) {
      return;
    }
    const svg = event.currentTarget as SVGSVGElement;
    const bounds = svg.getBoundingClientRect();
    const x = ((event.clientX - bounds.left) / bounds.width) * vbW - MAP_VIEW_PADDING;
    const y = ((event.clientY - bounds.top) / bounds.height) * vbH - MAP_VIEW_PADDING;
    const snapDistance = 42;
    let nearest: { id: number; distance: number } | null = null;
    for (const star of props.stars) {
      const dx = star.x - x;
      const dy = star.y - y;
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

  return (
    <svg
      class="galaxy-map"
      viewBox={viewBox}
      role="img"
      aria-label="Galaxy map"
      onMouseMove={handleMouseMove}
      onMouseLeave={() => props.onHoverSnappedStar?.(null)}
    >
      <defs>
        <linearGradient id="panelGlow" x1="0%" x2="100%" y1="0%" y2="100%">
          <stop offset="0%" stop-color="#0a1623" />
          <stop offset="100%" stop-color="#06101a" />
        </linearGradient>
      </defs>
      <rect
        x={-MAP_VIEW_PADDING}
        y={-MAP_VIEW_PADDING}
        width={vbW}
        height={vbH}
        rx="6"
        fill="url(#panelGlow)"
      />
      <g class="fleet-route-layer">
        <For each={props.routeLines ?? []}>
          {(route) => {
            return (
              <g classList={{ "is-preview": Boolean(route.preview) }}>
                <line x1={route.fromX} y1={route.fromY} x2={route.toX} y2={route.toY} class="fleet-route-line" />
                <text x={route.fromX + 10} y={route.fromY - 10} class="fleet-route-eta">
                  ETA {route.eta}
                </text>
              </g>
            );
          }}
        </For>
      </g>
      <g>
        <For each={props.stars}>
          {(star) => (
            <g
              class="galaxy-star"
              classList={{ selected: props.selectedStarId === star.id }}
              transform={`translate(${star.x} ${star.y})`}
              onClick={() => props.onSelect(star.id)}
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
                style={star.ownerColor ? { fill: star.ownerColor } : undefined}
              >
                {star.name}
              </text>
            </g>
          )}
        </For>
      </g>
      <g>
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
              <text text-anchor="middle" y="3.5">🚀</text>
            </g>
          )}
        </For>
      </g>
    </svg>
  );
}
