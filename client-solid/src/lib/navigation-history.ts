import { createSignal } from "solid-js";

const DEFAULT_PLANET_RETURN_PATH = "/galaxy";

const [planetReturnPath, setPlanetReturnPath] = createSignal<string>(
  DEFAULT_PLANET_RETURN_PATH,
);

export { planetReturnPath };

/**
 * Record the page the user is on so a later planet-page ESC can return to it.
 * Only `/galaxy` and `/colonies` are valid return targets; other paths fall
 * back to `/galaxy` on the next read.
 */
export function recordPlanetReturnCandidate(pathname: string) {
  if (pathname.startsWith("/galaxy")) {
    setPlanetReturnPath("/galaxy");
  } else if (pathname.startsWith("/colonies")) {
    setPlanetReturnPath("/colonies");
  }
}
