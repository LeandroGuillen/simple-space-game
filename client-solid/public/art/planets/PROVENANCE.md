# SSG planet art provenance

- Date: 2026-11-14.
- Tool/model: Codex CLI, gpt-6.1-sol, using the built-in Codex/Sol
  `image_gen.imagegen` image generation tool. Each variant was generated in its
  own image generation call. No external art, stock imagery, other generator,
  or manually drawn replacement was used.
- Delivery: three numbered variants per environment, 36 PNGs, 256×256 RGBA.
  Generated alpha is preserved. The generated rasters are resized for the game
  and saved with lossless PNG compression; this processing does not synthesize
  planet imagery. Transparent margins are cropped and the subject is fitted
  within 208×208 pixels, centered on the 256×256 canvas with Lanczos resizing.
- License: original SSG assets generated for this repository; permission is
  granted to use, copy, modify, and distribute them with SSG and its derivatives.
  Inspired by the look of 1990s 4X planet views, not a copy of any trademarked
  sprite or existing game's assets.

## Prompt summary

Original strategy-game sprites in a clean retro 1990s 4X painted style, with
restrained crisp details readable at 256px. Each subject is isolated and centered
on a square transparent canvas, occupying about 80% of the canvas, viewed with
consistent orthographic framing. Three-quarter light comes from the upper left;
the lower right falls into soft deep shadow. No stars, text, borders, logos, or
copied game assets. Planet types are single spheres; asteroid belts are elliptical
fields of irregular rocks with no central sphere.

Variants use different surface geography, cloud patterns, and palettes:

| Environment | Variant 1 | Variant 2 | Variant 3 |
| --- | --- | --- | --- |
| gas-giant | Amber/cream bands, russet storm | Teal/blue bands, white cyclone | Lavender/rose/ivory storms |
| asteroid-belt | Gray cratered rocks | Brown rocky asteroids | Charcoal/nickel cratered rocks |
| toxic | Acid green/sulfur storms | Mustard/olive corrosive world | Violet haze/chartreuse basins |
| radiated | Rust scars/violet rim | Purple rock/orange fissures | Copper gray/turquoise scars |
| barren | Gray craters/basalt plains | Umber craters/ejecta rays | Charcoal overlapping craters |
| desert | Golden dunes/rust ridges | Red sands/salt basins | Ochre sands/dark canyons |
| arid | Tan land/olive scrub | Rust highlands/river valleys | Buff plains/sage/shallow lakes |
| tundra | Snow/dark tundra | Icy continents/steel seas | Dark mountains/glaciers/moss |
| swamp | Olive marshes/muddy rivers | Emerald land/brown shallows | Moss/teal estuaries/peat |
| ocean | Cobalt seas/tiny islands | Cerulean/aqua volcanic islands | Indigo/teal sandy atolls |
| terran | Green continents/polar caps | Forests/sapphire seas | Temperate archipelagos/ice |
| gaia | Emerald land/turquoise seas | Jade land/azure inland seas | Verdant land/turquoise shelves |

The numbered filenames use the environment slugs from `src/lib/art.ts`.
[`generation.json`](generation.json) records each subject prompt summary,
generation file ID, source SHA-256, and delivered PNG SHA-256.
No old planet placeholders are retained. Ships, UI art, and Odin are outside
the scope of this release.
