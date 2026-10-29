import type { ApiStatusResponse } from "./types/api";

/** Whether the UI should offer the debug research-all control. */
export function isServerDebugEnabled(status: ApiStatusResponse | null | undefined): boolean {
  return status?.debug === true;
}
