import { ref } from "vue";
import { describe, expect, test, vi } from "vitest";
import { useLauncherKeyboard } from "./launcherKeyboard";

function keyEvent(key: string, values: Partial<KeyboardEvent> = {}): KeyboardEvent {
  return {
    type: "keydown",
    key,
    code: key,
    ctrlKey: false,
    shiftKey: false,
    altKey: false,
    metaKey: false,
    preventDefault: vi.fn(),
    stopPropagation: vi.fn(),
    ...values,
  } as KeyboardEvent;
}

function setup() {
  const target = {
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  };
  const input = { focus: vi.fn() } as unknown as HTMLInputElement;
  const options = {
    recordingHotkey: ref(false),
    settingsOpen: ref(false),
    query: ref(""),
    captureHotkey: vi.fn(),
    openSettings: vi.fn(async () => undefined),
    closeSettings: vi.fn(async () => undefined),
    hideLauncher: vi.fn(async () => undefined),
    moveSelection: vi.fn(),
    launchAtIndex: vi.fn(),
    launchSelected: vi.fn(),
    debug: vi.fn(async () => undefined),
    queryInput: () => input,
    target,
    activeElement: () => null,
  };
  return { keyboard: useLauncherKeyboard(options), options, target, input };
}

describe("launcher keyboard", () => {
  test("registers and removes global keyboard handlers", () => {
    const { keyboard, target } = setup();
    keyboard.mount();
    expect(target.addEventListener).toHaveBeenCalledWith("keydown", keyboard.onKeydown, true);
    keyboard.unmount();
    expect(target.removeEventListener).toHaveBeenCalledWith("keydown", keyboard.onKeydown, true);
  });

  test("dispatches navigation, indexed launch, launch action, and escape", () => {
    const { keyboard, options } = setup();
    keyboard.onKeydown(keyEvent("ArrowDown"));
    keyboard.onKeydown(keyEvent("2", { altKey: true, code: "Digit2" }));
    keyboard.onKeydown(keyEvent("Enter", { code: "Enter" }));
    keyboard.onKeydown(keyEvent("Escape"));

    expect(options.moveSelection).toHaveBeenCalledWith(1);
    expect(options.launchAtIndex).toHaveBeenCalledWith(1);
    expect(options.launchSelected).toHaveBeenCalledWith("code");
    expect(options.hideLauncher).toHaveBeenCalledOnce();
  });

  test("routes recording and text input before launcher actions", () => {
    const { keyboard, options, input } = setup();
    options.recordingHotkey.value = true;
    const recorded = keyEvent("K");
    keyboard.onKeydown(recorded);
    expect(options.captureHotkey).toHaveBeenCalledWith(recorded);

    options.recordingHotkey.value = false;
    keyboard.onKeydown(keyEvent("a"));
    expect(options.query.value).toBe("a");
    expect(input.focus).toHaveBeenCalledWith({ preventScroll: true });
  });
});
