import { errorDetail } from "./errors";
import type { CachedLauncherState, LynConfig, Project } from "./types";

const launcherCacheKey = "lyn-launcher-state-v1";

export function readLauncherCache(): CachedLauncherState | null {
  try {
    const raw = window.localStorage.getItem(launcherCacheKey);
    if (!raw) {
      return null;
    }
    const parsed: unknown = JSON.parse(raw);
    if (!isCachedLauncherState(parsed)) {
      return null;
    }
    return parsed;
  } catch (error) {
    reportLauncherCacheError(error);
    return null;
  }
}

export function writeLauncherCache(
  cfg: LynConfig | null,
  projects: Project[],
  projectIcons: Record<string, string>,
): void {
  if (!cfg) {
    return;
  }
  try {
    const state: CachedLauncherState = {
      version: 1,
      cfg,
      projects,
      projectIcons,
    };
    window.localStorage.setItem(launcherCacheKey, JSON.stringify(state));
  } catch (error) {
    reportLauncherCacheError(error);
  }
}

function reportLauncherCacheError(error: unknown): void {
  window.dispatchEvent(
    new CustomEvent("lyn-launcher-cache-error", {
      detail: errorDetail(error),
    }),
  );
}

function isCachedLauncherState(value: unknown): value is CachedLauncherState {
  if (!isRecord(value)) {
    return false;
  }
  return (
    value.version === 1 &&
    isLynConfig(value.cfg) &&
    Array.isArray(value.projects) &&
    value.projects.every(isProject) &&
    isStringRecord(value.projectIcons)
  );
}

function isLynConfig(value: unknown): value is LynConfig {
  if (!isRecord(value)) {
    return false;
  }
  const scanner = value.scanner;
  const ui = value.ui;
  const startup = value.startup;
  const hotkey = value.hotkey;
  const cache = value.cache;
  return (
    hasString(value, "path") &&
    isRecord(cache) &&
    hasString(cache, "dir") &&
    isRecord(startup) &&
    hasBoolean(startup, "enabled") &&
    hasBoolean(startup, "startHidden") &&
    isRecord(scanner) &&
    isStringArray(scanner.roots) &&
    hasNumber(scanner, "maxDepth") &&
    hasNumber(scanner, "concurrency") &&
    hasString(scanner, "timeout") &&
    hasBoolean(scanner, "watch") &&
    isRecord(hotkey) &&
    hasString(hotkey, "binding") &&
    isRecord(ui) &&
    hasString(ui, "theme") &&
    hasNumber(ui, "backgroundOpacity") &&
    hasString(ui, "selectionColor") &&
    hasString(ui, "windowPlacement") &&
    hasBoolean(ui, "clearQueryOnShow") &&
    hasString(ui, "workspaceQueryShortcut")
  );
}

function isProject(value: unknown): value is Project {
  if (!isRecord(value)) {
    return false;
  }
  return (
    hasString(value, "name") &&
    hasString(value, "path") &&
    hasString(value, "kind") &&
    hasString(value, "detectedAt") &&
    hasNumber(value, "usageCount") &&
    hasString(value, "lastLaunchedAt")
  );
}

function isStringRecord(value: unknown): value is Record<string, string> {
  if (!isRecord(value)) {
    return false;
  }
  return Object.values(value).every(isString);
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every(isString);
}

function hasString(value: Record<string, unknown>, key: string): boolean {
  return isString(value[key]);
}

function hasNumber(value: Record<string, unknown>, key: string): boolean {
  return typeof value[key] === "number";
}

function hasBoolean(value: Record<string, unknown>, key: string): boolean {
  return typeof value[key] === "boolean";
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}
