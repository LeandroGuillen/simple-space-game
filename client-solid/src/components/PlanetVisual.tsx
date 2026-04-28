import { cssClassForEnvironment } from "../lib/utils";
import type { JSX } from "solid-js";

type PlanetVisualProps = {
  environment: number;
  size?: "small" | "medium" | "large";
};

export function PlanetVisual(props: PlanetVisualProps): JSX.Element {
  const size = () => props.size ?? "large";
  const env = () => cssClassForEnvironment(props.environment);

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
      <div class={`${orbitClass()} ${orbitClass()}--${env()}`} />
    </div>
  );
}