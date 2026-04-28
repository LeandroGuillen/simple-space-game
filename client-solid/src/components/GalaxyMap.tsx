import { For } from "solid-js";

import type { StarSummary } from "../lib/types/domain";
import { STAR_COLORS } from "../lib/utils";

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
};

export function GalaxyMap(props: GalaxyMapProps) {
  const vbW = MAP_PIX_W + 2 * MAP_VIEW_PADDING;
  const vbH = MAP_PIX_H + 2 * MAP_VIEW_PADDING;
  const viewBox = `${-MAP_VIEW_PADDING} ${-MAP_VIEW_PADDING} ${vbW} ${vbH}`;
  return (
    <svg
      class="galaxy-map"
      viewBox={viewBox}
      role="img"
      aria-label="Galaxy map"
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
    </svg>
  );
}
