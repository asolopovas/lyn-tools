import type { ProjectAction } from "./types";

export type LauncherKeyEvent = Pick<
  KeyboardEvent,
  "altKey" | "code" | "ctrlKey" | "key" | "metaKey" | "shiftKey"
>;

export type HandledEvent = {
  preventDefault: () => void;
  stopPropagation: () => void;
};

export type LauncherHotkeyEvent = LauncherKeyEvent & HandledEvent;

export const launcherHotkeyEvents = ["keydown"] as const;

export function isEnterKey(event: LauncherKeyEvent): boolean {
  return (
    event.key === "Enter" ||
    event.key === "Return" ||
    event.key === "NumpadEnter" ||
    event.code === "Enter" ||
    event.code === "NumpadEnter"
  );
}

export function isArrowDownKey(event: LauncherKeyEvent): boolean {
  return event.key === "ArrowDown" || event.key === "Down";
}

export function isArrowUpKey(event: LauncherKeyEvent): boolean {
  return event.key === "ArrowUp" || event.key === "Up";
}

export function hasTextInputModifier(event: LauncherKeyEvent): boolean {
  return event.ctrlKey || event.altKey || event.metaKey;
}

export function launcherActionForKey(event: LauncherKeyEvent): ProjectAction | null {
  if (event.altKey) {
    return null;
  }
  if (isEnterKey(event)) {
    if (event.ctrlKey && event.shiftKey) {
      return "run-admin";
    }
    if (event.ctrlKey) {
      return "open";
    }
    if (event.shiftKey) {
      return "terminal";
    }
    return "code";
  }
  if (!event.ctrlKey || !event.shiftKey) {
    return null;
  }
  switch (event.key.toLowerCase()) {
    case "u":
      return "run-user";
    case "e":
      return "reveal";
    case "c":
      return "terminal";
    default:
      return null;
  }
}

export function launcherIndexForAltKey(event: LauncherKeyEvent): number | null {
  if (!event.altKey || event.ctrlKey || event.shiftKey || event.metaKey) {
    return null;
  }
  const keyNumber = Number.parseInt(event.key, 10);
  if (Number.isInteger(keyNumber) && keyNumber >= 1 && keyNumber <= 5) {
    return keyNumber - 1;
  }
  const codeMatch = /^(?:Digit|Numpad)([1-5])$/.exec(event.code);
  const codeNumber = codeMatch?.[1];
  if (codeNumber) {
    return Number.parseInt(codeNumber, 10) - 1;
  }
  return null;
}

export function consumeEvent(event: HandledEvent, stopPropagation = true): void {
  event.preventDefault();
  if (stopPropagation) {
    event.stopPropagation();
  }
}

export function triggerLauncherHotkey(
  event: LauncherHotkeyEvent,
  trigger: (action: ProjectAction) => void,
  stopPropagation = false,
): boolean {
  const action = launcherActionForKey(event);
  if (!action) {
    return false;
  }
  consumeEvent(event, stopPropagation);
  trigger(action);
  return true;
}
