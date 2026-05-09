import { describe, expect, it } from "vitest";

import {
  FULL_GALAXY_VIEWPORT,
  MAX_ZOOM,
  PAN_OVERSCROLL,
  WORLD_H,
  WORLD_W,
  WORLD_X,
  WORLD_Y,
  clampViewport,
  getZoom,
  parseStoredGalaxyViewport,
  screenToSvgPoint,
  zoomViewportAt,
} from "../galaxy-viewport";

describe("galaxy viewport", () => {
  it("clamps viewport inside world bounds with overscroll", () => {
    const clamped = clampViewport({
      x: -10000,
      y: 10000,
      width: WORLD_W / 2,
      height: WORLD_H / 2,
    });

    expect(clamped.x).toBe(WORLD_X - PAN_OVERSCROLL);
    expect(clamped.y).toBe(WORLD_Y + WORLD_H - clamped.height + PAN_OVERSCROLL);
  });

  it("keeps the zoom anchor stable while zooming in", () => {
    const bounds = { left: 0, top: 0, width: 900, height: 600 };
    const beforePoint = screenToSvgPoint(
      { clientX: 300, clientY: 240 },
      bounds,
      FULL_GALAXY_VIEWPORT,
    );
    const zoomed = zoomViewportAt(FULL_GALAXY_VIEWPORT, beforePoint, 1.22);
    const afterPoint = screenToSvgPoint(
      { clientX: 300, clientY: 240 },
      bounds,
      zoomed,
    );

    expect(afterPoint.x).toBeCloseTo(beforePoint.x, 5);
    expect(afterPoint.y).toBeCloseTo(beforePoint.y, 5);
  });

  it("does not zoom in past the maximum", () => {
    let viewport = FULL_GALAXY_VIEWPORT;
    for (let i = 0; i < 40; i += 1) {
      viewport = zoomViewportAt(viewport, { x: 350, y: 200 }, 1.22);
    }

    expect(getZoom(viewport)).toBeCloseTo(MAX_ZOOM, 5);
  });

  it("does not zoom out past the full galaxy", () => {
    const zoomed = zoomViewportAt(FULL_GALAXY_VIEWPORT, { x: 350, y: 200 }, 3);
    const zoomedOut = zoomViewportAt(zoomed, { x: 350, y: 200 }, 0.001);

    expect(zoomedOut.width).toBe(WORLD_W);
    expect(zoomedOut.height).toBe(WORLD_H);
    expect(getZoom(zoomedOut)).toBe(1);
  });

  it("uses the full galaxy viewport as reset state", () => {
    expect(FULL_GALAXY_VIEWPORT).toEqual({
      x: WORLD_X,
      y: WORLD_Y,
      width: WORLD_W,
      height: WORLD_H,
    });
  });

  it("loads valid stored viewport JSON", () => {
    const viewport = parseStoredGalaxyViewport(
      JSON.stringify({ x: 10, y: 20, width: 300, height: 200 }),
    );

    expect(viewport).toEqual({ x: 10, y: 20, width: 300, height: 200 });
  });

  it("falls back to the full galaxy for invalid stored viewport JSON", () => {
    expect(parseStoredGalaxyViewport("{")).toEqual(FULL_GALAXY_VIEWPORT);
    expect(parseStoredGalaxyViewport(JSON.stringify({ x: 0, width: 100 }))).toEqual(FULL_GALAXY_VIEWPORT);
  });

  it("converts hover coordinates through the active viewport", () => {
    const viewport = { x: 120, y: 80, width: 300, height: 200 };
    const point = screenToSvgPoint(
      { clientX: 150, clientY: 100 },
      { left: 0, top: 0, width: 300, height: 200 },
      viewport,
    );

    expect(point).toEqual({ x: 270, y: 180 });
  });

  it("accounts for preserved SVG aspect ratio letterboxing", () => {
    const point = screenToSvgPoint(
      { clientX: 200, clientY: 100 },
      { left: 0, top: 0, width: 400, height: 200 },
      FULL_GALAXY_VIEWPORT,
    );

    expect(point.x).toBeCloseTo(350, 5);
    expect(point.y).toBeCloseTo(200, 5);
  });
});
