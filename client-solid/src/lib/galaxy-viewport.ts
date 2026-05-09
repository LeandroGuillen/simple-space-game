export type GalaxyMapViewport = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export type ClientPoint = {
  clientX: number;
  clientY: number;
};

export const GALAXY_W = 700;
export const GALAXY_H = 400;
export const MAP_VIEW_PADDING = 100;

export const WORLD_X = -MAP_VIEW_PADDING;
export const WORLD_Y = -MAP_VIEW_PADDING;
export const WORLD_W = GALAXY_W + 2 * MAP_VIEW_PADDING;
export const WORLD_H = GALAXY_H + 2 * MAP_VIEW_PADDING;

export const MIN_ZOOM = 1;
export const MAX_ZOOM = 5;
export const ZOOM_STEP = 1.22;
export const PAN_OVERSCROLL = 60;

export const FULL_GALAXY_VIEWPORT: GalaxyMapViewport = {
  x: WORLD_X,
  y: WORLD_Y,
  width: WORLD_W,
  height: WORLD_H,
};

export const GALAXY_VIEWPORT_STORAGE_PREFIX = "client-solid.galaxyViewport.";

export function viewportToViewBox(viewport: GalaxyMapViewport) {
  return `${viewport.x} ${viewport.y} ${viewport.width} ${viewport.height}`;
}

export function getZoom(viewport: GalaxyMapViewport) {
  return WORLD_W / viewport.width;
}

export function isFiniteViewport(value: unknown): value is GalaxyMapViewport {
  if (!value || typeof value !== "object") {
    return false;
  }
  const viewport = value as Partial<GalaxyMapViewport>;
  return (
    Number.isFinite(viewport.x) &&
    Number.isFinite(viewport.y) &&
    Number.isFinite(viewport.width) &&
    Number.isFinite(viewport.height) &&
    (viewport.width ?? 0) > 0 &&
    (viewport.height ?? 0) > 0
  );
}

export function clampViewport(viewport: GalaxyMapViewport): GalaxyMapViewport {
  const minWidth = WORLD_W / MAX_ZOOM;
  const maxWidth = WORLD_W / MIN_ZOOM;
  const minHeight = WORLD_H / MAX_ZOOM;
  const maxHeight = WORLD_H / MIN_ZOOM;

  const width = clamp(viewport.width, minWidth, maxWidth);
  const height = clamp(viewport.height, minHeight, maxHeight);
  const minX = WORLD_X - PAN_OVERSCROLL;
  const maxX = WORLD_X + WORLD_W - width + PAN_OVERSCROLL;
  const minY = WORLD_Y - PAN_OVERSCROLL;
  const maxY = WORLD_Y + WORLD_H - height + PAN_OVERSCROLL;

  return {
    x: clamp(viewport.x, Math.min(minX, maxX), Math.max(minX, maxX)),
    y: clamp(viewport.y, Math.min(minY, maxY), Math.max(minY, maxY)),
    width,
    height,
  };
}

export function screenToSvgPoint(
  point: ClientPoint,
  bounds: Pick<DOMRect, "left" | "top" | "width" | "height">,
  viewport: GalaxyMapViewport,
) {
  const rendered = getRenderedViewportBounds(bounds, viewport);
  const relativeX = rendered.width > 0 ? (point.clientX - bounds.left - rendered.offsetX) / rendered.width : 0;
  const relativeY = rendered.height > 0 ? (point.clientY - bounds.top - rendered.offsetY) / rendered.height : 0;

  return {
    x: viewport.x + relativeX * viewport.width,
    y: viewport.y + relativeY * viewport.height,
  };
}

export function zoomViewportAt(
  viewport: GalaxyMapViewport,
  svgPoint: { x: number; y: number },
  scale: number,
) {
  const nextWidth = viewport.width / scale;
  const nextHeight = viewport.height / scale;
  const relativeX = (svgPoint.x - viewport.x) / viewport.width;
  const relativeY = (svgPoint.y - viewport.y) / viewport.height;

  return clampViewport({
    x: svgPoint.x - relativeX * nextWidth,
    y: svgPoint.y - relativeY * nextHeight,
    width: nextWidth,
    height: nextHeight,
  });
}

export function panViewportByScreenDelta(
  viewport: GalaxyMapViewport,
  bounds: Pick<DOMRect, "width" | "height">,
  deltaClientX: number,
  deltaClientY: number,
) {
  const rendered = getRenderedViewportBounds(bounds, viewport);
  const scaleX = rendered.width > 0 ? viewport.width / rendered.width : 0;
  const scaleY = rendered.height > 0 ? viewport.height / rendered.height : 0;

  return clampViewport({
    ...viewport,
    x: viewport.x - deltaClientX * scaleX,
    y: viewport.y - deltaClientY * scaleY,
  });
}

export function centerViewportOn(
  viewport: GalaxyMapViewport,
  point: { x: number; y: number },
) {
  return clampViewport({
    ...viewport,
    x: point.x - viewport.width / 2,
    y: point.y - viewport.height / 2,
  });
}

export function parseStoredGalaxyViewport(raw: string | null) {
  if (!raw) {
    return FULL_GALAXY_VIEWPORT;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    return isFiniteViewport(parsed) ? clampViewport(parsed) : FULL_GALAXY_VIEWPORT;
  } catch {
    return FULL_GALAXY_VIEWPORT;
  }
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}

function getRenderedViewportBounds(
  bounds: Pick<DOMRect, "width" | "height">,
  viewport: GalaxyMapViewport,
) {
  const viewportRatio = viewport.width / viewport.height;
  const boundsRatio = bounds.width / bounds.height;
  if (!Number.isFinite(viewportRatio) || !Number.isFinite(boundsRatio) || bounds.width <= 0 || bounds.height <= 0) {
    return { width: 0, height: 0, offsetX: 0, offsetY: 0 };
  }
  if (boundsRatio > viewportRatio) {
    const width = bounds.height * viewportRatio;
    return {
      width,
      height: bounds.height,
      offsetX: (bounds.width - width) / 2,
      offsetY: 0,
    };
  }
  const height = bounds.width / viewportRatio;
  return {
    width: bounds.width,
    height,
    offsetX: 0,
    offsetY: (bounds.height - height) / 2,
  };
}
