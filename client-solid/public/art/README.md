# Art Assets

Static game art lives here so Vite can serve it directly from stable `/art/...` URLs.

- `planets/`: original generated planet environment art keyed from `src/lib/art.ts`.
- `ships/`: ship sprites and animation placeholders keyed from `src/lib/art.ts`.
- `ui/`: shared UI bitmap placeholders.

Planet variants use numbered filenames such as `terran1.png`, `terran2.png`, and `barren3.png`. The client discovers variants at runtime by probing numbered files, so adding or removing variant files does not require a UI rebuild. Keep numbering within the configured probe limit in `src/lib/art.ts`.

The v0.25.0 planet set replaces all planet placeholders with three distinct variants
for each of the 12 environments (36 PNGs total). Every sprite is 256×256 RGBA,
with a transparent background, consistent upper-left three-quarter lighting, and
centered framing. Asteroid belts depict fields of rocks rather than spheres.
PNG compression keeps the raster assets small without losing alpha.

Generated for SSG with Codex/Sol image generation on 2026-11-14. See
[planet provenance](planets/PROVENANCE.md) for the tool, prompts, and license.
Ships and UI assets are unchanged by this planet art release.

Keep numbered planet filenames stable and within the discovery limit. The art
tests check every environment for at least three discoverable numbered PNGs,
the 256×256 dimensions, and an explicit alpha channel.
