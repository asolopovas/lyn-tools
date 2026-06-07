<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { EventsEmit, EventsOn, WindowCenter, WindowSetSize } from "./wailsRuntime";
import { backend } from "./backend";
import LauncherPanel from "./components/LauncherPanel.vue";
import SettingsPanel from "./components/SettingsPanel.vue";
import {
  consumeEvent,
  hasTextInputModifier,
  isArrowDownKey,
  isArrowUpKey,
  isEnterKey,
  launcherHotkeyEvents,
  launcherIndexForAltKey,
  triggerLauncherHotkey,
} from "./hotkeys";
import { useLauncherInput, scrollSelectedResultIntoView } from "./launcherInput";
import { useLauncherLaunch } from "./launcherLaunch";
import { useLauncherState } from "./launcherState";
import { useSettingsState } from "./settingsState";
import { themeByKey, themes } from "./themes";
const launcherWidth = 640;
const settingsWidth = 640;
const settingsHeight = 720;
const api = backend;
const launcher = useLauncherState(api);
const {
  query,
  cfg,
  projects,
  matches,
  projectIcons,
  selectedIndex,
  loading,
  scanning,
  status,
  selectedProject,
  statusLine,
  cacheState,
  ensureInitialProjects,
  loadConfigOnly,
  loadVisibleIcons,
  refreshForShow,
  refreshFromWatcher,
  scan,
  updateMatches,
  moveSelection,
} = launcher;
const appReady = ref(false);
const windowMode = ref<"launcher" | "settings">("launcher");
const settingsOpen = ref(false);
const settingsWindow = computed(() => windowMode.value === "settings");
const themeKeys = computed(() => Object.keys(themes));
const activeTheme = computed(() => themeByKey(cfg.value?.ui.theme ?? "power-run"));
const themeStyle = computed(() => ({
  "--lyn-bg": activeTheme.value.background,
  "--lyn-panel": activeTheme.value.panel,
  "--lyn-panel-alt": activeTheme.value.panelAlt,
  "--lyn-border": activeTheme.value.border,
  "--lyn-text": activeTheme.value.text,
  "--lyn-muted": activeTheme.value.muted,
  "--lyn-accent": activeTheme.value.accent,
  "--lyn-selected": activeTheme.value.selected,
  "--lyn-opacity": String(cfg.value?.ui.backgroundOpacity ?? 0.98),
}));
const launcherHeight = computed(() => (settingsOpen.value ? settingsHeight : 306));
const windowWidth = computed(() => (settingsOpen.value ? settingsWidth : launcherWidth));
const settings = useSettingsState({
  cfg,
  activeTheme,
  status,
  updateMatches,
  cacheState,
  closeSettings,
  api,
});
const {
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
} = settings;
const {
  launchAtIndex,
  launchDefault,
  launchProjectAction,
  launchSelected,
  updateNativeLaunchSelection,
} = useLauncherLaunch({
  api,
  settingsOpen,
  status,
  query,
  matches,
  selectedIndex,
  selectedProject,
  hideLauncher,
});
const { queryInput, focusQuery } = useLauncherInput();
let launchSelectionPoll = 0;
let hideOnBlurTimer = 0;
let eventDisposers: Array<() => void> = [];

watch(matches, () => {
  void loadVisibleIcons();
  updateNativeLaunchSelection();
});

watch(selectedProject, () => {
  scrollSelectedResultIntoView();
  updateNativeLaunchSelection();
});

watch(settingsOpen, (open) => {
  if (open) {
    exportTheme();
    void loadElevationStatus();
  }
  void nextTick(() => {
    placeLauncher();
    updateNativeLaunchSelection();
  });
});

watch(launcherHeight, (height) => {
  placeLauncher(height);
});

async function openSettings(): Promise<void> {
  if (settingsWindow.value) {
    return;
  }
  blurHideSuppressed.value = true;
  try {
    await api.OpenSettingsWindow();
  } catch (error) {
    status.value = error instanceof Error ? error.message : "Failed to open settings";
  } finally {
    window.setTimeout(() => {
      blurHideSuppressed.value = false;
    }, 700);
  }
}

async function closeSettings(): Promise<void> {
  if (settingsWindow.value) {
    await api.CloseSettingsWindow();
    return;
  }
  settingsOpen.value = false;
}

function prepareForShow(): void {
  if (cfg.value?.ui.clearQueryOnShow ?? true) {
    query.value = "";
    selectedIndex.value = 0;
  }
  focusQuery();
}

function placeLauncher(height = launcherHeight.value): void {
  WindowSetSize(windowWidth.value, height);
  WindowCenter();
}

async function hideLauncher(): Promise<void> {
  settingsOpen.value = false;
  await api.Hide();
}

function onWindowBlur(): void {
  if (hideOnBlurTimer) {
    window.clearTimeout(hideOnBlurTimer);
  }
  hideOnBlurTimer = window.setTimeout(() => {
    hideOnBlurTimer = 0;
    if (document.hasFocus() || blurHideSuppressed.value) {
      return;
    }
    void api.Debug("window.blur", "hide");
    void hideLauncher();
  }, 120);
}

function focusInput(input: HTMLInputElement): void {
  input.focus({ preventScroll: true });
}

function replaceQuery(update: (value: string) => string, input: HTMLInputElement): void {
  query.value = update(query.value);
  focusInput(input);
}

function consumeAndRun(event: KeyboardEvent, action: () => void, stopPropagation = true): void {
  consumeEvent(event, stopPropagation);
  action();
}

function onLauncherShown(...data: unknown[]): void {
  const showSeq = typeof data[0] === "number" ? data[0] : 0;
  void (async () => {
    settingsOpen.value = false;
    prepareForShow();
    if (!cfg.value || !projects.value.length) {
      await ensureInitialProjects();
    }
    placeLauncher();
    focusQuery();
    updateNativeLaunchSelection();
    EventsEmit("launcher-ready", showSeq);
    void refreshForShow();
  })();
}

function onKeydown(event: KeyboardEvent): void {
  if (recordingHotkey.value) {
    captureHotkey(event);
    return;
  }
  if (event.ctrlKey && event.key === ",") {
    consumeEvent(event, false);
    void openSettings();
    return;
  }
  if (event.key === "Escape") {
    consumeAndRun(event, () => {
      if (settingsOpen.value) {
        void closeSettings();
        return;
      }
      void hideLauncher();
    });
    return;
  }
  if (settingsOpen.value) {
    return;
  }
  if (isArrowDownKey(event)) {
    consumeAndRun(event, () => moveSelection(1));
    return;
  }
  if (isArrowUpKey(event)) {
    consumeAndRun(event, () => moveSelection(-1));
    return;
  }
  const altIndex = launcherIndexForAltKey(event);
  if (altIndex !== null) {
    consumeAndRun(event, () => launchAtIndex(altIndex));
    return;
  }
  if (isEnterKey(event)) {
    void api.Debug(
      "hotkey.event",
      `${event.type} key=${event.key} code=${event.code} ctrl=${event.ctrlKey} shift=${event.shiftKey} alt=${event.altKey} meta=${event.metaKey}`,
    );
  }
  if (triggerLauncherHotkey(event, launchSelected, true)) {
    return;
  }
  const input = queryInput();
  if (hasTextInputModifier(event)) {
    return;
  }
  if (!input || document.activeElement === input) {
    return;
  }
  if (event.key.length === 1) {
    consumeAndRun(event, () => replaceQuery((value) => value + event.key, input), false);
    return;
  }
  if (event.key === "Backspace" && query.value) {
    consumeAndRun(event, () => replaceQuery((value) => value.slice(0, -1), input), false);
  }
}

onMounted(() => {
  void (async () => {
    windowMode.value = await api.WindowMode();
    settingsOpen.value = settingsWindow.value;
    if (settingsWindow.value) {
      await loadConfigOnly();
      exportTheme();
      await loadElevationStatus();
      placeLauncher();
      appReady.value = true;
      for (const eventName of launcherHotkeyEvents) {
        window.addEventListener(eventName, onKeydown, true);
      }
      return;
    }
    appReady.value = true;
    void ensureInitialProjects();
    placeLauncher();
    focusQuery();
    for (const eventName of launcherHotkeyEvents) {
      window.addEventListener(eventName, onKeydown, true);
    }
    window.addEventListener("focus", focusQuery);
    window.addEventListener("blur", onWindowBlur);
    launchSelectionPoll = window.setInterval(updateNativeLaunchSelection, 100);
    updateNativeLaunchSelection();
    eventDisposers = [
      EventsOn("open-settings", () => {
        void openSettings();
      }),
      EventsOn("launcher-shown", onLauncherShown),
      EventsOn("projects-updated", () => {
        void refreshFromWatcher();
      }),
    ];
  })();
});

onUnmounted(() => {
  for (const eventName of launcherHotkeyEvents) {
    window.removeEventListener(eventName, onKeydown, true);
  }
  window.removeEventListener("focus", focusQuery);
  window.removeEventListener("blur", onWindowBlur);
  window.clearInterval(launchSelectionPoll);
  if (hideOnBlurTimer) {
    window.clearTimeout(hideOnBlurTimer);
  }
  void api.SetLaunchSelection({ path: "", action: "code" });
  for (const dispose of eventDisposers) {
    dispose();
  }
  eventDisposers = [];
});
</script>

<template>
  <main class="shell" :style="themeStyle">
    <section
      v-if="appReady"
      class="launcher"
      :class="{ 'settings-window': settingsOpen }"
      :style="{ width: `${windowWidth}px`, height: `${launcherHeight}px` }"
    >
      <SettingsPanel
        v-if="settingsOpen && cfg"
        v-model:cfg="cfg"
        v-model:root-draft="rootDraft"
        v-model:theme-draft="themeDraft"
        :theme-keys="themeKeys"
        :scanning="scanning"
        :elevation-status="elevationStatus"
        :recording-hotkey="recordingHotkey"
        @close="closeSettings"
        @save="saveSettings"
        @scan="scan"
        @export-theme="exportTheme"
        @import-theme="importTheme"
        @browse-root="browseRoot"
        @switch-elevation="switchElevation"
        @add-root="addRoot()"
        @remove-root="removeRoot"
        @normalize-workspace-shortcut="normalizeWorkspaceShortcut"
        @toggle-hotkey-recording="toggleHotkeyRecording"
        @capture-hotkey="captureHotkey"
      />
      <LauncherPanel
        v-else-if="!settingsOpen"
        v-model:query="query"
        :matches="matches"
        :selected-index="selectedIndex"
        :loading="loading"
        :status-line="statusLine"
        :scanning="scanning"
        :project-icons="projectIcons"
        @launch-selected="launchSelected"
        @open-settings="openSettings"
        @launch="launchDefault"
        @action="launchProjectAction"
        @select="selectedIndex = $event"
        @scan="scan"
      />
    </section>
  </main>
</template>
