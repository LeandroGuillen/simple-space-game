import { For } from 'solid-js'

import type { StarSummary } from '../lib/types/domain'
import { STAR_COLORS } from '../lib/utils'

type GalaxyMapProps = {
  stars: StarSummary[]
  selectedStarId?: number
  onSelect: (starId: number) => void
}

export function GalaxyMap(props: GalaxyMapProps) {
  return (
    <svg class="galaxy-map" viewBox="0 0 700 400" role="img" aria-label="Galaxy map">
      <defs>
        <linearGradient id="panelGlow" x1="0%" x2="100%" y1="0%" y2="100%">
          <stop offset="0%" stop-color="#101b2c" />
          <stop offset="100%" stop-color="#070b12" />
        </linearGradient>
      </defs>
      <rect x="0" y="0" width="700" height="400" rx="16" fill="url(#panelGlow)" />
      <g>
        <For each={props.stars}>
          {(star) => (
            <g
              class="galaxy-star"
              classList={{ selected: props.selectedStarId === star.id }}
              transform={`translate(${star.x} ${star.y})`}
              onClick={() => props.onSelect(star.id)}
            >
              <circle class="star-halo" r="10" fill={STAR_COLORS[star.color] ?? STAR_COLORS[0]} />
              <circle class="star-core" r="4.5" fill={STAR_COLORS[star.color] ?? STAR_COLORS[0]} />
              <circle class="star-ring" r="12" />
              <text x="-16" y="20">
                {star.name}
              </text>
            </g>
          )}
        </For>
      </g>
    </svg>
  )
}
