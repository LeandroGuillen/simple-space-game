import { onCleanup, onMount } from "solid-js";

const TYPING_TAGS = new Set(["INPUT", "TEXTAREA", "SELECT"]);

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  if (TYPING_TAGS.has(target.tagName)) {
    return true;
  }
  return target.isContentEditable;
}

function isModalOpen(): boolean {
  return Boolean(document.querySelector('[role="dialog"][aria-modal="true"]'));
}

export type ShortcutOptions = {
  /** Allow the shortcut to fire even when a modal dialog is open. */
  allowWhenModalOpen?: boolean;
  /** Additional gate; the shortcut only fires when this returns true. */
  when?: () => boolean;
};

/**
 * Bind a single-key shortcut to a callback. Skipped while the user is typing in
 * an input/textarea/contenteditable, while modifier keys are held, and (by
 * default) while a modal dialog is open.
 */
export function createShortcut(
  key: string,
  callback: (event: KeyboardEvent) => void,
  options: ShortcutOptions = {},
) {
  const targetKey = key.toLowerCase();
  onMount(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() !== targetKey) return;
      if (event.ctrlKey || event.metaKey || event.altKey) return;
      if (isTypingTarget(event.target)) return;
      if (!options.allowWhenModalOpen && isModalOpen()) return;
      if (options.when && !options.when()) return;
      callback(event);
    };
    document.addEventListener("keydown", handler);
    onCleanup(() => document.removeEventListener("keydown", handler));
  });
}

/** Append a uppercase keyboard-shortcut hint to a tooltip label. */
export function withShortcutHint(label: string, key: string): string {
  return `${label} (shortcut: ${key.toUpperCase()})`;
}
