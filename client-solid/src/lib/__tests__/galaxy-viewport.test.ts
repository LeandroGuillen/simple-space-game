import { describe, expect, it } from "vitest";

import {
  MAX_ZOOM,
  MIN_ZOOM,
  PAN_OVERSCROLL,
  clampViewport,
  getFullGalaxyViewport,
  getInitialGalaxyViewport,
  getWorldRect,
  getZoom,
  isAtMaxZoom,
  normalizeGalaxyMapBounds,
  panViewportByScreenDelta,
  parseStoredGalaxyViewport,
  screenToSvgPoint,
  zoomViewportAt,
} from "../galaxy-viewport";

describe("galaxy viewport", () => {
  const defaultBounds = { width: 700, height: 400, padding: 100 };
  const defaultWorld = getWorldRect(defaultBounds);
  const defaultFullViewport = getFullGalaxyViewport(defaultBounds);

  it("clamps viewport inside world bounds with overscroll", () => {
    const clamped = clampViewport({
      x: -10000,
      y: 10000,
      width: defaultWorld.width / 2,
      height: defaultWorld.height / 2,
    }, defaultBounds);

    expect(clamped.x).toBe(defaultWorld.x - PAN_OVERSCROLL);
    expect(clamped.y).toBe(defaultWorld.y + defaultWorld.height - clamped.height + PAN_OVERSCROLL);
  });

  it("builds a full viewport from huge galaxy bounds", () => {
    expect(getFullGalaxyViewport({ width: 1278, height: 730, padding: 100 })).toEqual({
      x: -100,
      y: -100,
      width: 1478,
      height: 930,
    });
  });

  it("starts huge galaxies zoomed out to 50%", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const hugeFullViewport = getFullGalaxyViewport(hugeBounds);
    const initialViewport = getInitialGalaxyViewport(hugeBounds);

    expect(getZoom(initialViewport, hugeBounds)).toBe(MIN_ZOOM);
    expect(initialViewport.width).toBe(hugeFullViewport.width / MIN_ZOOM);
    expect(initialViewport.height).toBe(hugeFullViewport.height / MIN_ZOOM);
    expect(initialViewport.x).toBe(hugeFullViewport.x + (hugeFullViewport.width - initialViewport.width) / 2);
    expect(initialViewport.y).toBe(hugeFullViewport.y + (hugeFullViewport.height - initialViewport.height) / 2);
  });

  it("starts medium galaxies at the full galaxy viewport", () => {
    const mediumBounds = { width: 990, height: 566, padding: 100 };

    expect(getInitialGalaxyViewport(mediumBounds)).toEqual(getFullGalaxyViewport(mediumBounds));
  });

  it("fits the initial viewport to the rendered star footprint", () => {
    const bounds = normalizeGalaxyMapBounds(250, 250, {
      minX: 42,
      minY: 58,
      maxX: 188,
      maxY: 214,
    });

    expect(getInitialGalaxyViewport(bounds)).toEqual({
      x: 32,
      y: 48,
      width: 166,
      height: 176,
    });
  });

  it("clamps against huge galaxy bounds", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const hugeWorld = getWorldRect(hugeBounds);
    const clamped = clampViewport({
      x: 9999,
      y: 9999,
      width: hugeWorld.width / 2,
      height: hugeWorld.height / 2,
    }, hugeBounds);

    expect(clamped.x).toBe(hugeWorld.x + hugeWorld.width - clamped.width + PAN_OVERSCROLL);
    expect(clamped.y).toBe(hugeWorld.y + hugeWorld.height - clamped.height + PAN_OVERSCROLL);
  });

  it("keeps the zoom anchor stable while zooming in", () => {
    const bounds = { left: 0, top: 0, width: 900, height: 600 };
    const beforePoint = screenToSvgPoint(
      { clientX: 300, clientY: 240 },
      bounds,
      defaultFullViewport,
    );
    const zoomed = zoomViewportAt(defaultFullViewport, beforePoint, 1.22, defaultBounds);
    const afterPoint = screenToSvgPoint(
      { clientX: 300, clientY: 240 },
      bounds,
      zoomed,
    );

    expect(afterPoint.x).toBeCloseTo(beforePoint.x, 5);
    expect(afterPoint.y).toBeCloseTo(beforePoint.y, 5);
  });

  it("does not zoom in past the maximum", () => {
    let viewport = defaultFullViewport;
    for (let i = 0; i < 40; i += 1) {
      viewport = zoomViewportAt(viewport, { x: 350, y: 200 }, 1.22, defaultBounds);
    }

    expect(getZoom(viewport, defaultBounds)).toBeCloseTo(MAX_ZOOM, 5);
  });

  it("ignores zoom-in actions once already at the maximum", () => {
    const maxZoomViewport = {
      ...defaultFullViewport,
      width: defaultWorld.width / MAX_ZOOM,
      height: defaultWorld.height / MAX_ZOOM,
    };

    expect(isAtMaxZoom(maxZoomViewport, defaultBounds)).toBe(true);
    expect(
      zoomViewportAt(maxZoomViewport, { x: 350, y: 200 }, 1.22, defaultBounds),
    ).toBe(maxZoomViewport);
  });

  it("allows zooming out past the full galaxy", () => {
    const zoomed = zoomViewportAt(defaultFullViewport, { x: 350, y: 200 }, 3, defaultBounds);
    const zoomedOut = zoomViewportAt(zoomed, { x: 350, y: 200 }, 0.001, defaultBounds);

    expect(zoomedOut.width).toBe(defaultWorld.width / MIN_ZOOM);
    expect(zoomedOut.height).toBe(defaultWorld.height / MIN_ZOOM);
    expect(getZoom(zoomedOut, defaultBounds)).toBe(MIN_ZOOM);
  });

  it("allows extra zoom-out steps for huge galaxies", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const hugeFullViewport = getFullGalaxyViewport(hugeBounds);
    const zoomedOutOnce = zoomViewportAt(hugeFullViewport, { x: 639, y: 365 }, 1 / 1.22, hugeBounds);
    const zoomedOutTwice = zoomViewportAt(zoomedOutOnce, { x: 639, y: 365 }, 1 / 1.22, hugeBounds);

    expect(getZoom(zoomedOutOnce, hugeBounds)).toBeLessThan(1);
    expect(getZoom(zoomedOutTwice, hugeBounds)).toBeLessThan(getZoom(zoomedOutOnce, hugeBounds));
    expect(zoomedOutTwice.width).toBeGreaterThan(hugeFullViewport.width);
  });

  it("does not pan when zooming out beyond the minimum zoom", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    let viewport = getFullGalaxyViewport(hugeBounds);
    for (let i = 0; i < 20; i += 1) {
      viewport = zoomViewportAt(viewport, { x: 1200, y: 700 }, 1 / 1.22, hugeBounds);
    }

    const before = viewport;
    const after = zoomViewportAt(before, { x: -100, y: -100 }, 1 / 1.22, hugeBounds);

    expect(getZoom(before, hugeBounds)).toBe(MIN_ZOOM);
    expect(after).toEqual(before);
  });

  it("keeps the galaxy centered at minimum zoom", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const hugeWorld = getWorldRect(hugeBounds);
    const zoomedOutViewport = {
      x: hugeWorld.x,
      y: hugeWorld.y,
      width: hugeWorld.width / MIN_ZOOM,
      height: hugeWorld.height / MIN_ZOOM,
    };
    const centeredX = hugeWorld.x + (hugeWorld.width - zoomedOutViewport.width) / 2;
    const centeredY = hugeWorld.y + (hugeWorld.height - zoomedOutViewport.height) / 2;

    expect(clampViewport({ ...zoomedOutViewport, x: -10000, y: -10000 }, hugeBounds)).toMatchObject({
      x: centeredX,
      y: centeredY,
    });
    expect(clampViewport({ ...zoomedOutViewport, x: 10000, y: 10000 }, hugeBounds)).toMatchObject({
      x: centeredX,
      y: centeredY,
    });
  });

  it("does not pan when the full galaxy fits the viewport", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const fullViewport = getFullGalaxyViewport(hugeBounds);

    const panned = panViewportByScreenDelta(
      fullViewport,
      hugeBounds,
      { width: 1200, height: 700 },
      200,
      120,
    );

    expect(getZoom(fullViewport, hugeBounds)).toBe(1);
    expect(panned).toEqual(fullViewport);
  });

  it("does not pan at minimum zoom", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const minZoomViewport = getInitialGalaxyViewport(hugeBounds);

    expect(panViewportByScreenDelta(
      minZoomViewport,
      hugeBounds,
      { width: 1200, height: 700 },
      200,
      120,
    )).toEqual(minZoomViewport);
  });

  it("uses the full galaxy viewport as reset state", () => {
    expect(defaultFullViewport).toEqual({
      x: defaultWorld.x,
      y: defaultWorld.y,
      width: defaultWorld.width,
      height: defaultWorld.height,
    });
  });

  it("loads valid stored viewport JSON", () => {
    const viewport = parseStoredGalaxyViewport(
      JSON.stringify({ x: 10, y: 20, width: 300, height: 200 }),
      defaultBounds,
    );

    expect(viewport).toEqual({ x: 10, y: 20, width: 300, height: 200 });
  });

  it("clamps stored viewport JSON to current bounds", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const hugeWorld = getWorldRect(hugeBounds);
    const viewport = parseStoredGalaxyViewport(
      JSON.stringify({ x: 9999, y: 9999, width: 300, height: 200 }),
      hugeBounds,
    );

    expect(viewport.x).toBe(hugeWorld.x + hugeWorld.width - viewport.width + PAN_OVERSCROLL);
    expect(viewport.y).toBe(hugeWorld.y + hugeWorld.height - viewport.height + PAN_OVERSCROLL);
  });

  it("falls back to the full galaxy for invalid stored viewport JSON", () => {
    expect(parseStoredGalaxyViewport("{", defaultBounds)).toEqual(defaultFullViewport);
    expect(parseStoredGalaxyViewport(JSON.stringify({ x: 0, width: 100 }), defaultBounds)).toEqual(defaultFullViewport);
  });

  it("falls back to the initial zoomed-out viewport for old huge games without stored viewport JSON", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };

    expect(parseStoredGalaxyViewport(null, hugeBounds)).toEqual(getInitialGalaxyViewport(hugeBounds));
  });

  it("uses current world width for zoom calculations", () => {
    const hugeBounds = { width: 1278, height: 730, padding: 100 };
    const hugeFullViewport = getFullGalaxyViewport(hugeBounds);

    expect(getZoom({ ...hugeFullViewport, width: hugeFullViewport.width / 2 }, hugeBounds)).toBe(2);
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
      defaultFullViewport,
    );

    expect(point.x).toBeCloseTo(350, 5);
    expect(point.y).toBeCloseTo(200, 5);
  });
});
