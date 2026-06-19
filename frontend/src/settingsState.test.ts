import { ref } from "vue";
import { describe, expect, it } from "vitest";
import { useSettingsState } from "./settingsState";
import type { LynConfig, WailsApp } from "./types";

function setup(wslReturn: unknown) {
  const api = { WSLDistros: async () => wslReturn } as unknown as WailsApp;
  return useSettingsState({
    cfg: ref<LynConfig | null>(null),
    status: ref(""),
    updateMatches: async () => {},
    cacheState: () => {},
    closeSettings: () => {},
    api,
  });
}

describe("loadWSLDistros", () => {
  it("coalesces a null backend result to an empty array", async () => {
    const state = setup(null);
    await state.loadWSLDistros();
    expect(state.wslDistros.value).toEqual([]);
  });

  it("keeps a returned distro list", async () => {
    const state = setup(["Ubuntu"]);
    await state.loadWSLDistros();
    expect(state.wslDistros.value).toEqual(["Ubuntu"]);
  });
});
