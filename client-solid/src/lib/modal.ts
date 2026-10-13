import { onCleanup, onMount } from "solid-js";

const openDialogs: HTMLElement[] = [];
let previousOverflow = "";

/** Attach inside a dialog's Solid ref so listeners follow its mounted lifetime. */
export function attachModalBehavior(dialog: HTMLElement, onClose: () => void) {
  const previousFocus = document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null;
  dialog.tabIndex = -1;
  const isTopDialog = () => openDialogs.at(-1) === dialog;
  const focusableElements = () => Array.from(dialog.querySelectorAll<HTMLElement>(
    'button, a[href], input, select, textarea, [tabindex]',
  )).filter((element) => element.tabIndex >= 0 && !element.matches(":disabled")
    && !element.closest('[hidden], [inert], [aria-hidden="true"]')
    && element.getClientRects().length > 0);
  const focusInside = () => (focusableElements()[0] ?? dialog).focus();

  const handleKeyDown = (event: KeyboardEvent) => {
    if (!isTopDialog()) return;
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopImmediatePropagation();
      onClose();
    } else if (event.key === "Tab") {
      const elements = focusableElements();
      const first = elements[0];
      const last = elements.at(-1);
      if (!first || !last) {
        event.preventDefault();
        dialog.focus();
      } else if (!dialog.contains(document.activeElement)
        || (event.shiftKey && document.activeElement === first)
        || (!event.shiftKey && document.activeElement === last)
        || document.activeElement === dialog) {
        event.preventDefault();
        (event.shiftKey ? last : first).focus();
      }
    }
  };
  const handleFocusIn = (event: FocusEvent) => {
    if (isTopDialog() && !dialog.contains(event.target as Node)) focusInside();
  };

  onMount(() => {
    if (openDialogs.length === 0) {
      previousOverflow = document.body.style.overflow;
      document.body.style.overflow = "hidden";
    }
    openDialogs.push(dialog);
    document.addEventListener("keydown", handleKeyDown, true);
    document.addEventListener("focusin", handleFocusIn);
    focusInside();
  });
  onCleanup(() => {
    const wasTopDialog = isTopDialog();
    const index = openDialogs.indexOf(dialog);
    if (index !== -1) openDialogs.splice(index, 1);
    document.removeEventListener("keydown", handleKeyDown, true);
    document.removeEventListener("focusin", handleFocusIn);
    if (openDialogs.length === 0 && index !== -1) {
      document.body.style.overflow = previousOverflow;
    }
    if (wasTopDialog) {
      const remainingDialog = openDialogs.at(-1);
      if (previousFocus?.isConnected && (!remainingDialog || remainingDialog.contains(previousFocus))) {
        previousFocus.focus();
      } else {
        remainingDialog?.focus();
      }
    }
  });
}
