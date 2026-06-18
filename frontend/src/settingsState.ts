import { ref } from "vue";
import { backend } from "./backend";
import { errorMessage } from "./errors";
import { consumeEvent } from "./hotkeys";
import { keyComboFromEvent } from "./hotkeyRecorder";
import type { ElevationMode, ElevationStatus, LynConfig, WailsApp, WSLRoot } from "./types";
import type { Ref } from "vue";

export function useSettingsState(options: {
  cfg: Ref<LynConfig | null>;
  status: Ref<string>;
  updateMatches: () => Promise<void>;
  cacheState: () => void;
  closeSettings: () => void | Promise<void>;
  api?: WailsApp;
}) {
  const api = options.api ?? backend;
  const wslDistros = ref<string[]>([]);
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

  function addRoot(path: string): void {
    const cfg = options.cfg.value;
    if (!cfg) {
      return;
    }
    const root = path.trim();
    if (!root || cfg.scanner.roots.includes(root)) {
      return;
    }
    cfg.scanner.roots.push(root);
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

  function removeWSLRoot(index: number): void {
    options.cfg.value?.scanner.wslRoots?.splice(index, 1);
  }

  async function browseWSLRoot(): Promise<void> {
    const cfg = options.cfg.value;
    if (!cfg) {
      return;
    }
    blurHideSuppressed.value = true;
    try {
      const root = await api.ChooseWSLFolder();
      if (root && root.path) {
        const list = (cfg.scanner.wslRoots ??= []);
        const exists = list.some(
          (item: WSLRoot) => (item.distro ?? "") === (root.distro ?? "") && item.path === root.path,
        );
        if (!exists) {
          list.push(root);
        }
      }
    } catch (error) {
      options.status.value = errorMessage(error, "Failed to add WSL folder");
    } finally {
      window.setTimeout(() => {
        blurHideSuppressed.value = false;
      }, 200);
    }
  }

  async function loadWSLDistros(): Promise<void> {
    try {
      wslDistros.value = await api.WSLDistros();
    } catch {
      wslDistros.value = [];
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
    wslDistros,
    elevationStatus,
    recordingHotkey,
    blurHideSuppressed,
    browseRoot,
    browseWSLRoot,
    removeWSLRoot,
    captureHotkey,
    loadElevationStatus,
    loadWSLDistros,
    normalizeWorkspaceShortcut,
    removeRoot,
    saveSettings,
    switchElevation,
    toggleHotkeyRecording,
  };
}
