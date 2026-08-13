import { nextTick, ref } from "vue";
import { describe, expect, test, vi } from "vitest";
import { readableTextColor, rgbaFromHex, useThemeState } from "./themeState";
import type { LynConfig } from "./types";

function config(): LynConfig {
  return {
    path: "",
    cache: { dir: "" },
    startup: { enabled: false, startHidden: false },
    scanner: {
      roots: [],
      maxDepth: 1,
      concurrency: 1,
      timeout: "1s",
      watch: false,
    },
    hotkey: { binding: "Ctrl+Space" },
    ui: {
      theme: "power-run",
      backgroundOpacity: 0.5,
      selectionColor: "#ffffff",
      windowPlacement: "center",
      clearQueryOnShow: true,
      workspaceQueryShortcut: "{",
    },
  };
}

describe("theme state", () => {
  test("derives colors and clamps opacity", () => {
    expect(rgbaFromHex("#204060", 1.5)).toBe("rgba(32, 64, 96, 1)");
    expect(rgbaFromHex("#204060", -1)).toBe("rgba(32, 64, 96, 0)");
    expect(rgbaFromHex("invalid", 0.5)).toBe("invalid");
    expect(readableTextColor("#ffffff")).toBe("#000000");
    expect(readableTextColor("#000000")).toBe("#ffffff");
  });

  test("synchronizes derived CSS variables when config changes", async () => {
    const cfg = ref<LynConfig | null>(config());
    const setProperty = vi.fn();
    const state = useThemeState(cfg, { setProperty });

    expect(state.surfaceColor.value).toBe("rgba(32, 32, 32, 0.5)");
    expect(setProperty).toHaveBeenCalledWith("--lyn-selected-text", "#000000");

    cfg.value!.ui.theme = "tron-legacy";
    cfg.value!.ui.backgroundOpacity = 0.25;
    cfg.value = { ...cfg.value! };
    await nextTick();

    expect(state.surfaceColor.value).toBe("rgba(7, 16, 22, 0.25)");
    expect(setProperty).toHaveBeenCalledWith("--lyn-surface", "rgba(7, 16, 22, 0.25)");
  });
});
