import { normalizeServerUrl } from "./utils";

const LAST_SETUP_SERVER_KEY = "client-solid.lastSetupServer";

/** Matches [`STORAGE_KEYS.serverUrl`](app-context.tsx) — last URL the app restored for API calls */
const LOCAL_STORAGE_SERVER_URL_KEY = "client-solid.serverUrl";

export function getSessionLastSetupServerUrl(): string {
  try {
    const raw = window.sessionStorage.getItem(LAST_SETUP_SERVER_KEY);
    return raw ? normalizeServerUrl(raw) : "";
  } catch {
    return "";
  }
}

export function setSessionLastSetupServerUrl(url: string) {
  try {
    const n = url.trim() ? normalizeServerUrl(url) : "";
    if (!n) {
      window.sessionStorage.removeItem(LAST_SETUP_SERVER_KEY);
      return;
    }
    window.sessionStorage.setItem(LAST_SETUP_SERVER_KEY, n);
  } catch {
    /* quota / private mode */
  }
}

export function clearSessionLastSetupServerUrl() {
  try {
    window.sessionStorage.removeItem(LAST_SETUP_SERVER_KEY);
  } catch {
    /* ignore */
  }
}

/** Prefer tab session (setup), then same key AppProvider uses so selection matches auto-connect */
export function getInitialSetupSelectedServerUrl(): string {
  if (typeof window === "undefined") {
    return "";
  }
  const fromSession = getSessionLastSetupServerUrl();
  if (fromSession) {
    return fromSession;
  }
  try {
    const fromLocal = window.localStorage.getItem(LOCAL_STORAGE_SERVER_URL_KEY) ?? "";
    return fromLocal ? normalizeServerUrl(fromLocal) : "";
  } catch {
    return "";
  }
}
