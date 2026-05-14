# Art Assets

Static game art lives here so Vite can serve it directly from stable `/art/...` URLs.

- `planets/`: planet environment art keyed from `src/lib/art.ts`.
- `ships/`: ship sprites and animation placeholders keyed from `src/lib/art.ts`.
- `ui/`: shared UI bitmap placeholders.

Planet variants use numbered filenames such as `terran1.png`, `terran2.png`, and `barren3.png`. The client discovers variants at runtime by probing numbered files, so adding or removing variant files does not require a UI rebuild. Keep numbering within the configured probe limit in `src/lib/art.ts`.

Replace placeholder files in place when final art is ready. Keep filenames stable unless you also update the mappings in `src/lib/art.ts`.
