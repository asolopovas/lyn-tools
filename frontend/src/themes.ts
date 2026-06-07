import type { Theme } from "./types";

export const themes: Record<string, Theme> = {
  "power-run": {
    name: "Power Run",
    background: "#202020",
    panel: "#1f1f1f",
    panelAlt: "#2d2d2d",
    border: "#515151",
    text: "#ffffff",
    muted: "#ffffff99",
    accent: "#60cdff",
    selected: "#333333",
  },
  "tron-legacy": {
    name: "Tron Legacy",
    background: "#071016",
    panel: "#0b151c",
    panelAlt: "#14242e",
    border: "#20d8ff66",
    text: "#d7f7ff",
    muted: "#8ab8c8",
    accent: "#ffb20d",
    selected: "#d7f7ff14",
  },
};

export function themeByKey(key: string): Theme {
  return themes[key] ?? themes["power-run"]!;
}

const themeFields = [
  "name",
  "background",
  "panel",
  "panelAlt",
  "border",
  "text",
  "muted",
  "accent",
  "selected",
] as const;

export function isTheme(value: unknown): value is Theme {
  if (!value || typeof value !== "object") {
    return false;
  }
  const item = value as Record<string, unknown>;
  return themeFields.every((key) => typeof item[key] === "string" && item[key].trim() !== "");
}
