import { computed, nextTick, ref, watch, type Ref } from "vue";
import { EventsEmit, EventsOn, WindowCenter, WindowSetSize } from "./wailsRuntime";
import type { LynConfig, Platform, Project, WailsApp, WindowMode } from "./types";

const launcherWidth = 640;
const settingsWidth = 720;
const settingsHeight = 650;
const launcherPanelHeight = 306;

type WindowHost = Pick<
  Window,
  | "addEventListener"
  | "removeEventListener"
  | "setInterval"
  | "clearInterval"
  | "setTimeout"
  | "clearTimeout"
>;

type RuntimeBridge = {
  emit: typeof EventsEmit;
  on: typeof EventsOn;
  center: typeof WindowCenter;
  setSize: typeof WindowSetSize;
};

export function useAppWindow(options: {
  api: WailsApp;
  cfg: Ref<LynConfig | null>;
  projects: Ref<Project[]>;
  query: Ref<string>;
  selectedIndex: Ref<number>;
  status: Ref<string>;
  settingsOpen?: Ref<boolean>;
  blurHideSuppressed: Ref<boolean>;
  ensureInitialProjects: () => Promise<void>;
  loadConfigOnly: () => Promise<void>;
  loadElevationStatus: () => Promise<void>;
  loadWSLDistros: () => Promise<void>;
  focusQuery: () => void;
  refreshForShow: () => Promise<void>;
  refreshFromWatcher: () => Promise<void>;
  updateNativeLaunchSelection: () => void;
  host?: WindowHost;
  hasFocus?: () => boolean;
  runtime?: RuntimeBridge;
}) {
  const host = options.host ?? window;
  const hasFocus = options.hasFocus ?? (() => document.hasFocus());
  const bridge = options.runtime ?? {
    emit: EventsEmit,
    on: EventsOn,
    center: WindowCenter,
    setSize: WindowSetSize,
  };
  const appReady = ref(false);
  const platform = ref<Platform>("");
  const windowMode = ref<WindowMode>("launcher");
  const settingsOpen = options.settingsOpen ?? ref(false);
  const settingsWindow = computed(() => windowMode.value === "settings");
  const launcherHeight = computed(() =>
    settingsOpen.value ? settingsHeight : launcherPanelHeight,
  );
  const windowWidth = computed(() => (settingsOpen.value ? settingsWidth : launcherWidth));
  let launchSelectionPoll = 0;
  let hideOnBlurTimer = 0;
  let eventDisposers: Array<() => void> = [];

  const stopSettingsWatch = watch(settingsOpen, (open) => {
    if (open) {
      void options.loadElevationStatus();
      void options.loadWSLDistros();
    }
    void nextTick(() => {
      placeLauncher();
      options.updateNativeLaunchSelection();
    });
  });
  const stopHeightWatch = watch(launcherHeight, (height) => {
    placeLauncher(height);
  });

  async function openSettings(): Promise<void> {
    if (settingsWindow.value) {
      return;
    }
    options.blurHideSuppressed.value = true;
    try {
      await options.api.OpenSettingsWindow();
    } catch (error) {
      options.status.value = error instanceof Error ? error.message : "Failed to open settings";
    } finally {
      host.setTimeout(() => {
        options.blurHideSuppressed.value = false;
      }, 700);
    }
  }

  async function closeSettings(): Promise<void> {
    if (settingsWindow.value) {
      await options.api.CloseSettingsWindow();
      return;
    }
    settingsOpen.value = false;
  }

  function prepareForShow(): void {
    if (options.cfg.value?.ui.clearQueryOnShow ?? true) {
      options.query.value = "";
      options.selectedIndex.value = 0;
    }
    options.focusQuery();
  }

  function placeLauncher(height = launcherHeight.value): void {
    bridge.setSize(windowWidth.value, height);
    bridge.center();
  }

  async function hideLauncher(): Promise<void> {
    settingsOpen.value = false;
    await options.api.Hide();
  }

  function onWindowBlur(): void {
    if (hideOnBlurTimer) {
      host.clearTimeout(hideOnBlurTimer);
    }
    hideOnBlurTimer = host.setTimeout(() => {
      hideOnBlurTimer = 0;
      if (hasFocus() || options.blurHideSuppressed.value) {
        return;
      }
      void options.api.Debug("window.blur", "hide");
      void hideLauncher();
    }, 120);
  }

  function onLauncherShown(...data: unknown[]): void {
    const showSequence = typeof data[0] === "number" ? data[0] : 0;
    void (async () => {
      settingsOpen.value = false;
      prepareForShow();
      if (!options.cfg.value || !options.projects.value.length) {
        await options.ensureInitialProjects();
      }
      placeLauncher();
      options.focusQuery();
      options.updateNativeLaunchSelection();
      bridge.emit("launcher-ready", showSequence);
      void options.refreshForShow();
    })();
  }

  async function mount(): Promise<void> {
    windowMode.value = await options.api.WindowMode();
    platform.value = await options.api.Platform();
    settingsOpen.value = settingsWindow.value;
    if (settingsWindow.value) {
      await options.loadConfigOnly();
      await options.loadElevationStatus();
      placeLauncher();
      appReady.value = true;
      return;
    }
    appReady.value = true;
    void options.ensureInitialProjects();
    placeLauncher();
    options.focusQuery();
    host.addEventListener("focus", options.focusQuery);
    host.addEventListener("blur", onWindowBlur);
    launchSelectionPoll = host.setInterval(options.updateNativeLaunchSelection, 100);
    options.updateNativeLaunchSelection();
    eventDisposers = [
      bridge.on("launcher-shown", onLauncherShown),
      bridge.on("projects-updated", () => {
        void options.refreshFromWatcher();
      }),
    ];
  }

  function unmount(): void {
    host.removeEventListener("focus", options.focusQuery);
    host.removeEventListener("blur", onWindowBlur);
    if (launchSelectionPoll) {
      host.clearInterval(launchSelectionPoll);
      launchSelectionPoll = 0;
    }
    if (hideOnBlurTimer) {
      host.clearTimeout(hideOnBlurTimer);
      hideOnBlurTimer = 0;
    }
    void options.api.SetLaunchSelection({ path: "", action: "code" });
    for (const dispose of eventDisposers) {
      dispose();
    }
    eventDisposers = [];
    stopSettingsWatch();
    stopHeightWatch();
  }

  return {
    appReady,
    platform,
    windowMode,
    settingsOpen,
    settingsWindow,
    launcherHeight,
    windowWidth,
    openSettings,
    closeSettings,
    hideLauncher,
    placeLauncher,
    onWindowBlur,
    onLauncherShown,
    mount,
    unmount,
  };
}
