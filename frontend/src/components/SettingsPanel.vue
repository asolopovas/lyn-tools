<script setup lang="ts">
import { computed } from "vue";
import { consumeEvent } from "../hotkeys";
import { icons } from "../icons";
import { themeByKey } from "../themes";
import type { ElevationMode, ElevationStatus, LynConfig } from "../types";

const cfg = defineModel<LynConfig>("cfg", { required: true });
const rootDraft = defineModel<string>("rootDraft", { required: true });
const themeDraft = defineModel<string>("themeDraft", { required: true });

const props = defineProps<{
  themeKeys: string[];
  scanning: boolean;
  elevationStatus: ElevationStatus | null;
  recordingHotkey: boolean;
}>();

const emit = defineEmits<{
  close: [];
  save: [];
  scan: [];
  "export-theme": [];
  "import-theme": [];
  "browse-root": [];
  "switch-elevation": [mode: ElevationMode];
  "add-root": [];
  "remove-root": [index: number];
  "normalize-workspace-shortcut": [];
  "toggle-hotkey-recording": [];
  "capture-hotkey": [event: KeyboardEvent];
}>();

const opacityPercent = computed(() => `${Math.round(cfg.value.ui.backgroundOpacity * 100)}%`);
const standardModeSelected = computed(() => props.elevationStatus?.mode !== "admin");
const adminModeSelected = computed(() => props.elevationStatus?.mode === "admin");

function addRootFromInput(event: KeyboardEvent): void {
  consumeEvent(event, false);
  emit("add-root");
}
</script>

<template>
  <div class="settings-panel">
    <header class="settings-topbar">
      <div class="settings-brand">
        <svg class="settings-brand-icon" viewBox="0 0 24 24" aria-hidden="true">
          <path :d="icons.settings" />
        </svg>
        <h1>Lyn Settings</h1>
      </div>
      <button
        class="settings-icon-button"
        type="button"
        title="Close settings"
        @click="$emit('close')"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.close" /></svg>
      </button>
    </header>

    <main class="settings-content">
      <section class="settings-section">
        <h2>Appearance</h2>
        <div class="settings-card">
          <label class="settings-field settings-card-row">
            <span>Theme</span>
            <select v-model="cfg.ui.theme">
              <option v-for="key in themeKeys" :key="key" :value="key">
                {{ themeByKey(key).name }}
              </option>
            </select>
          </label>
          <label class="settings-field settings-card-row">
            <span class="settings-row-title"
              ><span>Opacity</span><code>{{ opacityPercent }}</code></span
            >
            <input
              v-model.number="cfg.ui.backgroundOpacity"
              type="range"
              min="0.55"
              max="1"
              step="0.01"
            />
          </label>
          <label class="settings-field settings-card-row">
            <span>Launcher Position</span>
            <select v-model="cfg.ui.windowPlacement">
              <option value="center">Center</option>
            </select>
          </label>
          <label class="settings-toggle-row settings-card-row">
            <span>Clear input when launcher opens</span>
            <span class="modern-switch">
              <input v-model="cfg.ui.clearQueryOnShow" type="checkbox" />
              <span class="modern-switch-track" aria-hidden="true"></span>
            </span>
          </label>
        </div>
      </section>

      <section class="settings-section">
        <h2>Hotkeys</h2>
        <div class="settings-card">
          <button
            class="settings-action-row"
            :class="{ recording: recordingHotkey }"
            type="button"
            @click="$emit('toggle-hotkey-recording')"
            @keydown="$emit('capture-hotkey', $event)"
          >
            <span class="settings-row-label"
              ><span class="settings-symbol">⌨</span>Global shortcut</span
            >
            <code>{{ recordingHotkey ? "Press keys" : cfg.hotkey.binding }}</code>
          </button>
          <label class="settings-action-row workspace-trigger-row">
            <span class="settings-row-label"
              ><span class="settings-symbol">⠿</span>Workspace trigger</span
            >
            <input
              v-model="cfg.ui.workspaceQueryShortcut"
              maxlength="1"
              spellcheck="false"
              @input="$emit('normalize-workspace-shortcut')"
            />
          </label>
        </div>
      </section>

      <section class="settings-section">
        <h2>Process Mode</h2>
        <div class="mode-card-grid">
          <button
            class="process-card"
            :class="{ selected: standardModeSelected }"
            type="button"
            :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'standard'"
            @click="$emit('switch-elevation', 'standard')"
          >
            <span class="process-glyph">♙</span>
            <span class="process-radio" aria-hidden="true"></span>
            <strong>Standard Mode</strong>
            <small
              >Runs with standard user privileges. Safer, but cannot interact with elevated
              apps.</small
            >
          </button>
          <button
            class="process-card admin"
            :class="{ selected: adminModeSelected }"
            type="button"
            :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'admin'"
            @click="$emit('switch-elevation', 'admin')"
          >
            <span class="process-glyph">♢</span>
            <span class="process-radio" aria-hidden="true"></span>
            <strong>Administrative Mode</strong>
            <small
              >Requires UAC. Can interact with all applications, including elevated ones.</small
            >
          </button>
          <small class="mode-message">{{
            elevationStatus?.message ?? "Checking process mode"
          }}</small>
        </div>
      </section>

      <section class="settings-section">
        <div class="settings-section-title-row">
          <h2>Indexing</h2>
          <div class="settings-section-actions">
            <button
              class="settings-link-button"
              type="button"
              :disabled="scanning"
              @click="$emit('scan')"
            >
              {{ scanning ? "Scanning" : "Scan" }}
            </button>
            <button class="settings-link-button" type="button" @click="$emit('browse-root')">
              + Add Folder
            </button>
          </div>
        </div>
        <div class="settings-card">
          <div v-for="(root, index) in cfg.scanner.roots" :key="root" class="root-row">
            <span class="root-path"
              ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.folder" /></svg
              >{{ root }}</span
            >
            <button type="button" title="Remove folder" @click="$emit('remove-root', index)">
              ⌫
            </button>
          </div>
          <div class="root-add-row">
            <input
              v-model="rootDraft"
              spellcheck="false"
              placeholder="Folder path, /home/... or \\wsl.localhost\..."
              @keydown.enter="addRootFromInput"
            />
            <button type="button" @click="$emit('add-root')">Add</button>
          </div>
          <label class="settings-toggle-row settings-card-row inset-row">
            <span
              ><span>Watch indexed folders</span
              ><small>Automatically update index on file changes</small></span
            >
            <span class="modern-switch">
              <input v-model="cfg.scanner.watch" type="checkbox" />
              <span class="modern-switch-track" aria-hidden="true"></span>
            </span>
          </label>
        </div>
      </section>

      <section class="settings-section">
        <h2>Startup</h2>
        <div class="settings-card">
          <label class="settings-toggle-row settings-card-row">
            <span>Start Lyn with the system</span>
            <span class="modern-switch">
              <input v-model="cfg.startup.enabled" type="checkbox" />
              <span class="modern-switch-track" aria-hidden="true"></span>
            </span>
          </label>
          <label class="settings-toggle-row settings-card-row">
            <span>Keep Lyn running in the background</span>
            <span class="modern-switch">
              <input
                v-model="cfg.startup.startHidden"
                type="checkbox"
                :disabled="!cfg.startup.enabled"
              />
              <span class="modern-switch-track" aria-hidden="true"></span>
            </span>
          </label>
        </div>
      </section>

      <section class="settings-section theme-json-section">
        <h2>Theme JSON</h2>
        <div class="theme-json-card">
          <textarea v-model="themeDraft" spellcheck="false" />
        </div>
      </section>
    </main>

    <nav class="settings-bottom-bar">
      <button type="button" @click="$emit('export-theme')"><span>↥</span>Export</button>
      <button type="button" @click="$emit('import-theme')"><span>↧</span>Import</button>
      <button type="button" @click="$emit('close')"><span>×</span>Cancel</button>
      <button class="settings-save-button" type="button" @click="$emit('save')">
        <span>✓</span>Save
      </button>
    </nav>
  </div>
</template>
