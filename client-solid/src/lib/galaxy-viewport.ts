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

export type GalaxyMapBounds = {
  x?: number;
  y?: number;
  width: number;
  height: number;
  padding: number;
};

export const DEFAULT_GALAXY_W = 70;
export const DEFAULT_GALAXY_H = 40;
export const MAP_VIEW_PADDING = 10;

export const MIN_ZOOM = 0.5;
export const MAX_ZOOM = 5;
export const ZOOM_STEP = 1.3;
export const PAN_OVERSCROLL = 60;
export const MIN_GALAXY_VIEW_SIZE = 120;

export const GALAXY_VIEWPORT_STORAGE_PREFIX = "client-solid.galaxyViewport.";

export function normalizeGalaxyMapBounds(
  width?: number,
  height?: number,
  content?: {
    minX: number;
    minY: number;
    maxX: number;
    maxY: number;
  },
): GalaxyMapBounds {
  if (
    content &&
    Number.isFinite(content.minX) &&
    Number.isFinite(content.minY) &&
    Number.isFinite(content.maxX) &&
    Number.isFinite(content.maxY) &&
    content.maxX >= content.minX &&
    content.maxY >= content.minY
  ) {
    const contentWidth = Math.max(
      content.maxX - content.minX,
      MIN_GALAXY_VIEW_SIZE,
    );
    const contentHeight = Math.max(
      content.maxY - content.minY,
      MIN_GALAXY_VIEW_SIZE,
    );
    const centerX = (content.minX + content.maxX) / 2;
    const centerY = (content.minY + content.maxY) / 2;
    return {
      x: centerX - contentWidth / 2,
      y: centerY - contentHeight / 2,
      width: contentWidth,
      height: contentHeight,
      padding: MAP_VIEW_PADDING,
    };
  }

  return {
    x: 0,
    y: 0,
    width: isPositiveFiniteNumber(width) ? width : DEFAULT_GALAXY_W,
    height: isPositiveFiniteNumber(height) ? height : DEFAULT_GALAXY_H,
    padding: MAP_VIEW_PADDING,
  };
}

export function getWorldRect(bounds: GalaxyMapBounds) {
  return {
    x: (bounds.x ?? 0) - bounds.padding,
    y: (bounds.y ?? 0) - bounds.padding,
    width: bounds.width + 2 * bounds.padding,
    height: bounds.height + 2 * bounds.padding,
  };
}

export function getFullGalaxyViewport(
  bounds: GalaxyMapBounds,
): GalaxyMapViewport {
  const world = getWorldRect(bounds);
  return {
    x: world.x,
    y: world.y,
    width: world.width,
    height: world.height,
  };
}

export function getInitialGalaxyViewport(
  bounds: GalaxyMapBounds,
): GalaxyMapViewport {
  const fullViewport = getFullGalaxyViewport(bounds);
  if (!shouldStartZoomedOut(bounds)) {
    return fullViewport;
  }

  const width = fullViewport.width / MIN_ZOOM;
  const height = fullViewport.height / MIN_ZOOM;
  return {
    x: fullViewport.x + (fullViewport.width - width) / 2,
    y: fullViewport.y + (fullViewport.height - height) / 2,
    width,
    height,
  };
}

export function viewportToViewBox(viewport: GalaxyMapViewport) {
  return `${viewport.x} ${viewport.y} ${viewport.width} ${viewport.height}`;
}

export function getZoom(viewport: GalaxyMapViewport, bounds: GalaxyMapBounds) {
  return getWorldRect(bounds).width / viewport.width;
}

export function isAtOrBeyondFullView(
  viewport: GalaxyMapViewport,
  bounds: GalaxyMapBounds,
) {
  const world = getWorldRect(bounds);
  return viewport.width >= world.width && viewport.height >= world.height;
}

export function isAtMinZoom(
  viewport: GalaxyMapViewport,
  bounds: GalaxyMapBounds,
) {
  return getZoom(viewport, bounds) <= MIN_ZOOM;
}

export function isAtMaxZoom(
  viewport: GalaxyMapViewport,
  bounds: GalaxyMapBounds,
) {
  return getZoom(viewport, bounds) >= MAX_ZOOM;
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

export function clampViewport(
  viewport: GalaxyMapViewport,
  bounds: GalaxyMapBounds,
): GalaxyMapViewport {
  const world = getWorldRect(bounds);
  const minWidth = world.width / MAX_ZOOM;
  const maxWidth = world.width / MIN_ZOOM;
  const minHeight = world.height / MAX_ZOOM;
  const maxHeight = world.height / MIN_ZOOM;

  const width = clamp(viewport.width, minWidth, maxWidth);
  const height = clamp(viewport.height, minHeight, maxHeight);
  const [minX, maxX] = getPanLimits(world.x, world.width, width);
  const [minY, maxY] = getPanLimits(world.y, world.height, height);

  return {
    x: clamp(viewport.x, minX, maxX),
    y: clamp(viewport.y, minY, maxY),
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
  const relativeX =
    rendered.width > 0
      ? (point.clientX - bounds.left - rendered.offsetX) / rendered.width
      : 0;
  const relativeY =
    rendered.height > 0
      ? (point.clientY - bounds.top - rendered.offsetY) / rendered.height
      : 0;

  return {
    x: viewport.x + relativeX * viewport.width,
    y: viewport.y + relativeY * viewport.height,
  };
}

export function zoomViewportAt(
  viewport: GalaxyMapViewport,
  svgPoint: { x: number; y: number },
  scale: number,
  bounds: GalaxyMapBounds,
) {
  if (scale > 1 && isAtMaxZoom(viewport, bounds)) {
    return viewport;
  }

  const world = getWorldRect(bounds);
  const maxWidth = world.width / MIN_ZOOM;
  const maxHeight = world.height / MIN_ZOOM;
  if (scale < 1 && viewport.width >= maxWidth && viewport.height >= maxHeight) {
    return clampViewport(viewport, bounds);
  }

  const nextWidth = viewport.width / scale;
  const nextHeight = viewport.height / scale;
  const relativeX = (svgPoint.x - viewport.x) / viewport.width;
  const relativeY = (svgPoint.y - viewport.y) / viewport.height;

  return clampViewport(
    {
      x: svgPoint.x - relativeX * nextWidth,
      y: svgPoint.y - relativeY * nextHeight,
      width: nextWidth,
      height: nextHeight,
    },
    bounds,
  );
}

export function panViewportByScreenDelta(
  viewport: GalaxyMapViewport,
  mapBounds: GalaxyMapBounds,
  screenBounds: Pick<DOMRect, "width" | "height">,
  deltaClientX: number,
  deltaClientY: number,
) {
  if (isAtMinZoom(viewport, mapBounds)) {
    return clampViewport(viewport, mapBounds);
  }

  const rendered = getRenderedViewportBounds(screenBounds, viewport);
  const scaleX = rendered.width > 0 ? viewport.width / rendered.width : 0;
  const scaleY = rendered.height > 0 ? viewport.height / rendered.height : 0;

  return clampViewport(
    {
      ...viewport,
      x: viewport.x - deltaClientX * scaleX,
      y: viewport.y - deltaClientY * scaleY,
    },
    mapBounds,
  );
}

export function centerViewportOn(
  viewport: GalaxyMapViewport,
  point: { x: number; y: number },
  bounds: GalaxyMapBounds,
) {
  return clampViewport(
    {
      ...viewport,
      x: point.x - viewport.width / 2,
      y: point.y - viewport.height / 2,
    },
    bounds,
  );
}

export function parseStoredGalaxyViewport(
  raw: string | null,
  bounds: GalaxyMapBounds,
) {
  const initialViewport = getInitialGalaxyViewport(bounds);
  if (!raw) {
    return initialViewport;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    return isFiniteViewport(parsed)
      ? clampViewport(parsed, bounds)
      : initialViewport;
  } catch {
    return initialViewport;
  }
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}

function isPositiveFiniteNumber(value: unknown): value is number {
  return Number.isFinite(value) && Number(value) > 0;
}

function shouldStartZoomedOut(bounds: GalaxyMapBounds) {
  return bounds.width >= 1200 || bounds.height >= 700;
}

function getPanLimits(
  worldStart: number,
  worldSize: number,
  viewportSize: number,
) {
  if (viewportSize < worldSize) {
    return [
      worldStart - PAN_OVERSCROLL,
      worldStart + worldSize - viewportSize + PAN_OVERSCROLL,
    ];
  }

  const centeredStart = worldStart + (worldSize - viewportSize) / 2;
  return [centeredStart, centeredStart];
}

function getRenderedViewportBounds(
  bounds: Pick<DOMRect, "width" | "height">,
  viewport: GalaxyMapViewport,
) {
  const viewportRatio = viewport.width / viewport.height;
  const boundsRatio = bounds.width / bounds.height;
  if (
    !Number.isFinite(viewportRatio) ||
    !Number.isFinite(boundsRatio) ||
    bounds.width <= 0 ||
    bounds.height <= 0
  ) {
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
