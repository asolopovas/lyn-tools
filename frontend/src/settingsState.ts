import { ref } from "vue";
import { backend } from "./backend";
import { errorMessage } from "./errors";
import { consumeEvent } from "./hotkeys";
import { keyComboFromEvent } from "./hotkeyRecorder";
import { isTheme, themes } from "./themes";
import type { ElevationMode, ElevationStatus, LynConfig, Theme, WailsApp } from "./types";
import type { ComputedRef, Ref } from "vue";

export function useSettingsState(options: {
  cfg: Ref<LynConfig | null>;
  activeTheme: ComputedRef<Theme>;
  status: Ref<string>;
  updateMatches: () => Promise<void>;
  cacheState: () => void;
  closeSettings: () => void | Promise<void>;
  api?: WailsApp;
}) {
  const api = options.api ?? backend;
  const rootDraft = ref("");
  const themeDraft = ref("");
  const elevationStatus = ref<ElevationStatus | null>(null);
  const recordingHotkey = ref(false);
  const blurHideSuppressed = ref(false);

  async function saveSettings(): Promise<void> {
    if (!options.cfg.value) {
      return;
    }
    options.cfg.value = await api.SaveConfig(options.cfg.value);
    await options.updateMatches();
    options.cacheState();
    await options.closeSettings();
    options.status.value = "Settings saved";
  }

  function addRoot(path = rootDraft.value): void {
    const cfg = options.cfg.value;
    if (!cfg) {
      return;
    }
    const root = path.trim();
    if (!root || cfg.scanner.roots.includes(root)) {
      rootDraft.value = "";
      return;
    }
    cfg.scanner.roots.push(root);
    rootDraft.value = "";
  }

  function removeRoot(index: number): void {
    options.cfg.value?.scanner.roots.splice(index, 1);
  }

  async function browseRoot(): Promise<void> {
    blurHideSuppressed.value = true;
    try {
      const root = await api.ChooseFolder();
      if (root) {
        addRoot(root);
      }
    } finally {
      window.setTimeout(() => {
        blurHideSuppressed.value = false;
      }, 200);
    }
  }

  async function loadElevationStatus(): Promise<void> {
    try {
      elevationStatus.value = await api.ElevationStatus();
    } catch (error) {
      options.status.value = errorMessage(error, "Failed to read elevation mode");
    }
  }

  async function switchElevation(mode: ElevationMode): Promise<void> {
    blurHideSuppressed.value = true;
    options.status.value = mode === "admin" ? "Waiting for UAC" : "Switching to standard mode";
    try {
      elevationStatus.value = await api.SwitchElevation(mode);
    } catch (error) {
      options.status.value = errorMessage(error, "Failed to switch mode");
      await loadElevationStatus();
      window.setTimeout(() => {
        blurHideSuppressed.value = false;
      }, 200);
    }
  }

  function normalizeWorkspaceShortcut(): void {
    const cfg = options.cfg.value;
    if (!cfg) {
      return;
    }
    cfg.ui.workspaceQueryShortcut = cfg.ui.workspaceQueryShortcut.trim().slice(0, 1) || "{";
  }

  function exportTheme(): void {
    themeDraft.value = JSON.stringify(options.activeTheme.value, null, 2);
  }

  function importTheme(): void {
    try {
      const imported: unknown = JSON.parse(themeDraft.value);
      if (!isTheme(imported)) {
        options.status.value = "Theme JSON is missing required fields";
        return;
      }
      const key = imported.name.toLowerCase().replaceAll(" ", "-");
      themes[key] = imported;
      if (options.cfg.value) {
        options.cfg.value.ui.theme = key;
      }
      options.status.value = "Theme imported";
    } catch {
      options.status.value = "Theme JSON is invalid";
    }
  }

  function startHotkeyRecording(): void {
    recordingHotkey.value = true;
    options.status.value = "Press shortcut";
  }

  function stopHotkeyRecording(): void {
    recordingHotkey.value = false;
    options.status.value = "Ready";
  }

  function toggleHotkeyRecording(): void {
    if (recordingHotkey.value) {
      stopHotkeyRecording();
      return;
    }
    startHotkeyRecording();
  }

  function captureHotkey(event: KeyboardEvent): void {
    if (!recordingHotkey.value || !options.cfg.value) {
      return;
    }
    consumeEvent(event);
    const combo = keyComboFromEvent(event);
    if (!combo) {
      options.status.value = "Shortcut needs a modifier and key";
      return;
    }
    options.cfg.value.hotkey.binding = combo;
    recordingHotkey.value = false;
    options.status.value = `Shortcut set to ${combo}`;
  }

  return {
    rootDraft,
    themeDraft,
    elevationStatus,
    recordingHotkey,
    blurHideSuppressed,
    addRoot,
    browseRoot,
    captureHotkey,
    exportTheme,
    importTheme,
    loadElevationStatus,
    normalizeWorkspaceShortcut,
    removeRoot,
    saveSettings,
    switchElevation,
    toggleHotkeyRecording,
  };
}
