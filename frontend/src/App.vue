<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useAppWindow } from "./appWindow";
import { backend } from "./backend";
import LauncherPanel from "./components/LauncherPanel.vue";
import SettingsPanel from "./components/SettingsPanel.vue";
import { useLauncherInput, scrollSelectedResultIntoView } from "./launcherInput";
import { useLauncherKeyboard } from "./launcherKeyboard";
import { useLauncherLaunch } from "./launcherLaunch";
import { useLauncherState } from "./launcherState";
import { useSettingsState } from "./settingsState";
import { useThemeState } from "./themeState";

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
const settingsOpen = ref(false);
const { themeKeys } = useThemeState(cfg);
const settings = useSettingsState({
  cfg,
  status,
  updateMatches,
  cacheState,
  closeSettings,
  api,
});
const {
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
} = settings;
const wslPresent = computed(
  () => wslDistros.value.length > 0 || (cfg.value?.scanner.wslRoots?.length ?? 0) > 0,
);
const { queryInput, focusQuery } = useLauncherInput();
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
const appWindow = useAppWindow({
  api,
  cfg,
  projects,
  query,
  selectedIndex,
  status,
  settingsOpen,
  blurHideSuppressed,
  ensureInitialProjects,
  loadConfigOnly,
  loadElevationStatus,
  loadWSLDistros,
  focusQuery,
  refreshForShow,
  refreshFromWatcher,
  updateNativeLaunchSelection,
});
const { appReady, platform, settingsWindow, launcherHeight } = appWindow;
const keyboard = useLauncherKeyboard({
  recordingHotkey,
  settingsOpen,
  query,
  captureHotkey,
  openSettings,
  closeSettings,
  hideLauncher,
  moveSelection,
  launchAtIndex,
  launchSelected,
  debug: api.Debug,
  queryInput,
});

watch(matches, () => {
  void loadVisibleIcons();
  updateNativeLaunchSelection();
});

watch(selectedProject, () => {
  scrollSelectedResultIntoView();
  updateNativeLaunchSelection();
});

async function openSettings(): Promise<void> {
  await appWindow.openSettings();
}

async function closeSettings(): Promise<void> {
  await appWindow.closeSettings();
}

async function hideLauncher(): Promise<void> {
  await appWindow.hideLauncher();
}

onMounted(() => {
  void (async () => {
    await appWindow.mount();
    keyboard.mount();
  })();
});

onUnmounted(() => {
  keyboard.unmount();
  appWindow.unmount();
});
</script>

<template>
  <main class="min-h-screen box-border bg-transparent">
    <section
      v-if="appReady"
      class="relative box-border w-full overflow-hidden [--wails-draggable:no-drag]"
      :class="settingsOpen ? 'block bg-bg' : 'rounded-md border border-line bg-surface'"
      :style="{
        width: '100%',
        height: `min(${launcherHeight}px, 100vh)`,
      }"
    >
      <SettingsPanel
        v-if="settingsOpen && cfg"
        v-model:cfg="cfg"
        :theme-keys="themeKeys"
        :scanning="scanning"
        :platform="platform"
        :wsl-present="wslPresent"
        :elevation-status="elevationStatus"
        :recording-hotkey="recordingHotkey"
        @close="closeSettings"
        @save="saveSettings"
        @scan="scan"
        @browse-root="browseRoot"
        @browse-wsl-root="browseWSLRoot"
        @switch-elevation="switchElevation"
        @remove-root="removeRoot"
        @remove-wsl-root="removeWSLRoot"
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
        :platform="platform"
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
