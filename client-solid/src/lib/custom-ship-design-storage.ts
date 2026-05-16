import type { ShipDesignForm } from "./types/domain";

const CUSTOM_SHIP_DRAFT_KEY = "client-solid.customShipDesignDraft";

export function getCustomShipDesignDraft(): ShipDesignForm | null {
  try {
    const raw = window.sessionStorage.getItem(CUSTOM_SHIP_DRAFT_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<ShipDesignForm>;
    const name = typeof parsed.name === "string" ? parsed.name : "";
    const drive = typeof parsed.drive === "string" ? parsed.drive : "";
    const computer = typeof parsed.computer === "string" ? parsed.computer : "";
    const shield = typeof parsed.shield === "string" ? parsed.shield : "";
    const armor = typeof parsed.armor === "string" ? parsed.armor : "";
    const hullSize = typeof parsed.hullSize === "string" ? parsed.hullSize : "";
    const fuelCell = typeof parsed.fuelCell === "string" ? parsed.fuelCell : "";
    return { name, drive, computer, shield, armor, hullSize, fuelCell };
  } catch {
    return null;
  }
}

export function setCustomShipDesignDraft(design: ShipDesignForm) {
  window.sessionStorage.setItem(CUSTOM_SHIP_DRAFT_KEY, JSON.stringify(design));
}

export function clearCustomShipDesignDraft() {
  window.sessionStorage.removeItem(CUSTOM_SHIP_DRAFT_KEY);
}
