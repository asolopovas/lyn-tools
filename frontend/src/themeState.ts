import { computed, watch, type Ref } from "vue";
import { themeByKey, themes } from "./themes";
import type { LynConfig } from "./types";

type StyleTarget = {
  setProperty: (name: string, value: string) => void;
};

export function rgbaFromHex(hex: string, opacity: number): string {
  const match = /^#([\da-f]{6})$/i.exec(hex);
  if (!match) {
    return hex;
  }
  const value = Number.parseInt(match[1]!, 16);
  const red = (value >> 16) & 255;
  const green = (value >> 8) & 255;
  const blue = value & 255;
  const alpha = Math.min(Math.max(opacity, 0), 1);
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}

export function readableTextColor(color: string): "#000000" | "#ffffff" {
  const match = /^#([\da-f]{6})$/i.exec(color);
  if (!match) {
    return "#ffffff";
  }
  const value = Number.parseInt(match[1]!, 16);
  const red = (value >> 16) & 255;
  const green = (value >> 8) & 255;
  const blue = value & 255;
  const luminance = (0.299 * red + 0.587 * green + 0.114 * blue) / 255;
  return luminance > 0.55 ? "#000000" : "#ffffff";
}

export function useThemeState(cfg: Ref<LynConfig | null>, target?: StyleTarget) {
  const themeKeys = computed(() => Object.keys(themes));
  const activeTheme = computed(() => themeByKey(cfg.value?.ui.theme ?? "power-run"));
  const selectedColor = computed(() => cfg.value?.ui.selectionColor || activeTheme.value.selected);
  const selectedTextColor = computed(() => readableTextColor(selectedColor.value));
  const backgroundOpacity = computed(() => cfg.value?.ui.backgroundOpacity ?? 0.98);
  const surfaceColor = computed(() =>
    rgbaFromHex(activeTheme.value.background, backgroundOpacity.value),
  );
  const themeStyle = computed(() => ({
    "--lyn-bg": activeTheme.value.background,
    "--lyn-panel": activeTheme.value.panel,
    "--lyn-panel-alt": activeTheme.value.panelAlt,
    "--lyn-border": activeTheme.value.border,
    "--lyn-text": activeTheme.value.text,
    "--lyn-muted": activeTheme.value.muted,
    "--lyn-accent": activeTheme.value.accent,
    "--lyn-selected": selectedColor.value,
    "--lyn-selected-text": selectedTextColor.value,
    "--lyn-opacity": String(backgroundOpacity.value),
    "--lyn-surface": surfaceColor.value,
  }));
  const styleTarget = target ?? document.documentElement.style;

  watch(
    themeStyle,
    (style) => {
      for (const [name, value] of Object.entries(style)) {
        styleTarget.setProperty(name, String(value));
      }
    },
    { immediate: true },
  );

  return {
    themeKeys,
    activeTheme,
    selectedColor,
    selectedTextColor,
    backgroundOpacity,
    surfaceColor,
    themeStyle,
  };
}
