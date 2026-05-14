import {
  discoverPlanetArtForEnvironment,
  planetArtForEnvironment,
  planetArtFallbackForEnvironment,
} from "../lib/art";
import { cssClassForEnvironment } from "../lib/utils";
import { createEffect, createSignal, onCleanup } from "solid-js";
import type { JSX } from "solid-js";

import "./PlanetVisual.css";

type PlanetVisualProps = {
  planetId?: number;
  environment: number;
  size?: "small" | "medium" | "large";
};

export function PlanetVisual(props: PlanetVisualProps): JSX.Element {
  const [artSrc, setArtSrc] = createSignal(
    planetArtFallbackForEnvironment(props.environment),
  );
  const size = () => props.size ?? "large";
  const env = () => cssClassForEnvironment(props.environment);

  createEffect(() => {
    const environment = props.environment;
    const planetId = props.planetId;
    let isCurrent = true;

    setArtSrc(planetArtFallbackForEnvironment(environment));

    discoverPlanetArtForEnvironment(environment).then((variants) => {
      if (isCurrent) {
        setArtSrc(planetArtForEnvironment(environment, planetId, variants));
      }
    });

    onCleanup(() => {
      isCurrent = false;
    });
  });

  const containerClass = () => {
    switch (size()) {
      case "small": return "planet-mini";
      case "medium": return "planet-visual-sm";
      case "large": return "planet-visual";
    }
  };

  const orbitClass = () => {
    switch (size()) {
      case "small": return "planet-mini";
      case "medium":
      case "large": return "planet-orbit";
    }
  };

  return (
    <div class={containerClass()}>
      <div class={`${orbitClass()} ${orbitClass()}--${env()}`}>
        <img class="planet-art" src={artSrc()} alt="" loading="lazy" />
      </div>
    </div>
  );
}
