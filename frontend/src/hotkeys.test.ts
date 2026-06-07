import { describe, expect, it, vi } from "vitest";
import {
  launcherActionForKey,
  launcherIndexForAltKey,
  triggerLauncherHotkey,
  type LauncherKeyEvent,
} from "./hotkeys";

function key(event: Partial<LauncherKeyEvent>): LauncherKeyEvent {
  return {
    altKey: false,
    code: "",
    ctrlKey: false,
    key: "",
    metaKey: false,
    shiftKey: false,
    ...event,
  };
}

describe("launcherActionForKey", () => {
  it.each([
    [key({ key: "Enter" }), "code"],
    [key({ key: "Enter", ctrlKey: true }), "open"],
    [key({ key: "Enter", shiftKey: true }), "terminal"],
    [key({ key: "Enter", ctrlKey: true, shiftKey: true }), "run-admin"],
    [key({ code: "NumpadEnter", key: "NumpadEnter" }), "code"],
    [key({ key: "Return" }), "code"],
    [key({ key: "Unidentified", code: "Enter" }), "code"],
    [key({ key: "u", ctrlKey: true, shiftKey: true }), "run-user"],
    [key({ key: "E", ctrlKey: true, shiftKey: true }), "reveal"],
    [key({ key: "c", ctrlKey: true, shiftKey: true }), "terminal"],
  ] as const)("maps %o to %s", (event, action) => {
    expect(launcherActionForKey(event)).toBe(action);
  });

  it("keeps Enter valid when Win is still reported after Win+D", () => {
    expect(launcherActionForKey(key({ key: "Enter", metaKey: true }))).toBe("code");
  });

  it("ignores Alt-modified hotkeys", () => {
    expect(launcherActionForKey(key({ key: "Enter", altKey: true }))).toBeNull();
  });
});

describe("launcherIndexForAltKey", () => {
  it.each([
    [key({ key: "1", altKey: true }), 0],
    [key({ key: "5", altKey: true }), 4],
    [key({ key: "Unidentified", code: "Digit3", altKey: true }), 2],
    [key({ key: "Unidentified", code: "Numpad4", altKey: true }), 3],
  ] as const)("maps %o to result index %s", (event, index) => {
    expect(launcherIndexForAltKey(event)).toBe(index);
  });

  it("ignores non-result Alt shortcuts", () => {
    expect(launcherIndexForAltKey(key({ key: "6", altKey: true }))).toBeNull();
    expect(launcherIndexForAltKey(key({ key: "1" }))).toBeNull();
    expect(launcherIndexForAltKey(key({ key: "1", altKey: true, ctrlKey: true }))).toBeNull();
  });
});

describe("triggerLauncherHotkey", () => {
  it("prevents the native event and triggers one launcher action", () => {
    const preventDefault = vi.fn();
    const stopPropagation = vi.fn();
    const trigger = vi.fn();
    const handled = triggerLauncherHotkey(
      { ...key({ key: "Enter" }), preventDefault, stopPropagation },
      trigger,
      true,
    );
    expect(handled).toBe(true);
    expect(preventDefault).toHaveBeenCalledOnce();
    expect(stopPropagation).toHaveBeenCalledOnce();
    expect(trigger).toHaveBeenCalledExactlyOnceWith("code");
  });

  it("returns false without touching unrelated key events", () => {
    const preventDefault = vi.fn();
    const trigger = vi.fn();
    const handled = triggerLauncherHotkey(
      { ...key({ key: "x" }), preventDefault, stopPropagation: vi.fn() },
      trigger,
    );
    expect(handled).toBe(false);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(trigger).not.toHaveBeenCalled();
  });
});
