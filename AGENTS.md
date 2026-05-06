# Repository Guidelines

## Project Structure & Module Organization

- `server/`: Go module `ssg`, with the HTTP entry point in `cmd/server/main.go`.
- `server/game/`: core game rules, turn processing, resource loading, and unit-tested domain logic.
- `server/model/`: shared game data types and generated enum string helpers.
- `server/handler/` and `server/ws/`: REST handlers and WebSocket plumbing.
- `server/game/resources/`: JSON data files for techs, buildings, equipment, planets, and names.
- `client-solid/`: Solid frontend. Source lives in `src/`, static assets in `public/`, and utility tests in `src/lib/__tests__/`.

## Build, Test, and Development Commands

Run backend commands from `server/`:

- `go run ./cmd/server`: start the API and WebSocket server.
- `go test ./...`: run all Go tests.
- `go fmt ./...`: format Go code before committing.

Run frontend commands from `client-solid/`:

- `npm install`: install frontend dependencies.
- `npm run dev`: start the Vite dev server.
- `npm run build`: type-check and build the frontend.
- `npm test`: run Vitest tests once.
- `npm run preview`: preview the production build locally.

## Coding Style & Naming Conventions

Use `gofmt` formatting for Go. Keep package names short and lowercase, and name Go tests `*_test.go` beside the code they cover. Prefer existing domain terms such as `Planet`, `ShipDesign`, `Turn`, and `ResourceManager`.

Frontend code uses TypeScript, Solid components, and CSS organized by feature or layout. Name components in `PascalCase` (`GalaxyMap.tsx`) and helpers in `camelCase`. Keep route components under `src/routes/` and shared API or mapping logic in `src/lib/`.

## Testing Guidelines

Backend tests use Go's `testing` package with `testify`. Add tests near changed game logic, especially for turn resolution, population, galaxy generation, and ship utilities.

Frontend tests use Vitest. Place tests in `__tests__` directories or use `*.test.ts` naming for new utility tests. Run both backend and frontend tests when changes touch API contracts.

## Commit & Pull Request Guidelines

Recent commits use short, lowercase summaries, for example `building in colonies` and `food surplus UI`. Keep the first line focused on behavior changed.

Pull requests should include a brief description, commands run, linked issues if applicable, and screenshots or recordings for visible UI changes. Note resource JSON changes because they can affect game balance.

## Agent-Specific Instructions

Do not rewrite generated enum string files unless the enum source changes. Treat `server/game/resources/*.json` as gameplay data: keep edits minimal, valid JSON, and covered by loader or behavior tests where practical.
