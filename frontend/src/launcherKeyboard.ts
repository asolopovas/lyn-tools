import type { Ref } from "vue";
import {
  consumeEvent,
  hasTextInputModifier,
  isArrowDownKey,
  isArrowUpKey,
  isEnterKey,
  launcherHotkeyEvents,
  launcherIndexForAltKey,
  triggerLauncherHotkey,
} from "./hotkeys";
import type { ProjectAction } from "./types";

type KeyboardTarget = Pick<Window, "addEventListener" | "removeEventListener">;

export function useLauncherKeyboard(options: {
  recordingHotkey: Ref<boolean>;
  settingsOpen: Ref<boolean>;
  query: Ref<string>;
  captureHotkey: (event: KeyboardEvent) => void;
  openSettings: () => Promise<void>;
  closeSettings: () => Promise<void>;
  hideLauncher: () => Promise<void>;
  moveSelection: (delta: number) => void;
  launchAtIndex: (index: number) => void;
  launchSelected: (action: ProjectAction) => void;
  debug: (stage: string, detail: string) => Promise<void>;
  queryInput: () => HTMLInputElement | null;
  target?: KeyboardTarget;
  activeElement?: () => Element | null;
}) {
  const target = options.target ?? window;
  const activeElement = options.activeElement ?? (() => document.activeElement);

  function focusInput(input: HTMLInputElement): void {
    input.focus({ preventScroll: true });
  }

  function replaceQuery(update: (value: string) => string, input: HTMLInputElement): void {
    options.query.value = update(options.query.value);
    focusInput(input);
  }

  function consumeAndRun(event: KeyboardEvent, action: () => void, stopPropagation = true): void {
    consumeEvent(event, stopPropagation);
    action();
  }

  function onKeydown(event: KeyboardEvent): void {
    if (options.recordingHotkey.value) {
      options.captureHotkey(event);
      return;
    }
    if (event.ctrlKey && event.key === ",") {
      consumeEvent(event, false);
      void options.openSettings();
      return;
    }
    if (event.key === "Escape") {
      consumeAndRun(event, () => {
        if (options.settingsOpen.value) {
          void options.closeSettings();
          return;
        }
        void options.hideLauncher();
      });
      return;
    }
    if (options.settingsOpen.value) {
      return;
    }
    if (isArrowDownKey(event)) {
      consumeAndRun(event, () => options.moveSelection(1));
      return;
    }
    if (isArrowUpKey(event)) {
      consumeAndRun(event, () => options.moveSelection(-1));
      return;
    }
    const altIndex = launcherIndexForAltKey(event);
    if (altIndex !== null) {
      consumeAndRun(event, () => options.launchAtIndex(altIndex));
      return;
    }
    if (isEnterKey(event)) {
      void options.debug(
        "hotkey.event",
        `${event.type} key=${event.key} code=${event.code} ctrl=${event.ctrlKey} shift=${event.shiftKey} alt=${event.altKey} meta=${event.metaKey}`,
      );
    }
    if (triggerLauncherHotkey(event, options.launchSelected, true)) {
      return;
    }
    const input = options.queryInput();
    if (hasTextInputModifier(event)) {
      return;
    }
    if (!input || activeElement() === input) {
      return;
    }
    if (event.key.length === 1) {
      consumeAndRun(event, () => replaceQuery((value) => value + event.key, input), false);
      return;
    }
    if (event.key === "Backspace" && options.query.value) {
      consumeAndRun(event, () => replaceQuery((value) => value.slice(0, -1), input), false);
    }
  }

  function mount(): void {
    for (const eventName of launcherHotkeyEvents) {
      target.addEventListener(eventName, onKeydown as EventListener, true);
    }
  }

  function unmount(): void {
    for (const eventName of launcherHotkeyEvents) {
      target.removeEventListener(eventName, onKeydown as EventListener, true);
    }
  }

  return { onKeydown, mount, unmount };
}
