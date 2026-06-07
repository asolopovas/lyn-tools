import { describe, expect, it } from "vitest";
import { keyComboFromEvent, normalizedHotkeyKey } from "./hotkeyRecorder";

function event(input: Partial<KeyboardEvent>): KeyboardEvent {
  return {
    altKey: false,
    code: "",
    ctrlKey: false,
    key: "",
    metaKey: false,
    shiftKey: false,
    ...input,
  } as KeyboardEvent;
}

describe("hotkey recorder", () => {
  it("normalizes supported keys", () => {
    expect(normalizedHotkeyKey(event({ key: "a" }))).toBe("A");
    expect(normalizedHotkeyKey(event({ code: "Space", key: " " }))).toBe("Space");
    expect(normalizedHotkeyKey(event({ key: "Enter" }))).toBe("Enter");
  });

  it("requires a modifier for recorded combinations", () => {
    expect(keyComboFromEvent(event({ key: "D", metaKey: true }))).toBe("Win+D");
    expect(keyComboFromEvent(event({ key: "D" }))).toBeNull();
  });
});
