<script setup lang="ts">
import { computed } from "vue";
import { icons } from "../icons";
import { themeByKey } from "../themes";
import type { ElevationMode, ElevationStatus, LynConfig } from "../types";

const cfg = defineModel<LynConfig>("cfg", { required: true });

const props = defineProps<{
  themeKeys: string[];
  scanning: boolean;
  wslPresent: boolean;
  elevationStatus: ElevationStatus | null;
  recordingHotkey: boolean;
}>();

defineEmits<{
  close: [];
  save: [];
  scan: [];
  "browse-root": [];
  "browse-wsl-root": [];
  "switch-elevation": [mode: ElevationMode];
  "remove-root": [index: number];
  "remove-wsl-root": [index: number];
  "normalize-workspace-shortcut": [];
  "toggle-hotkey-recording": [];
  "capture-hotkey": [event: KeyboardEvent];
}>();

const opacityPercent = computed(() => `${Math.round(cfg.value.ui.backgroundOpacity * 100)}%`);
const standardModeSelected = computed(() => props.elevationStatus?.mode !== "admin");
const adminModeSelected = computed(() => props.elevationStatus?.mode === "admin");
</script>

<template>
  <div class="settings-panel">
    <header class="settings-topbar">
      <div class="settings-brand">
        <svg class="settings-brand-icon" viewBox="0 0 24 24" aria-hidden="true">
          <path :d="icons.settings" />
        </svg>
        <h1>Settings</h1>
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
          <label class="settings-field settings-card-row color-field-row">
            <span>Highlight color</span>
            <input v-model="cfg.ui.selectionColor" type="color" />
          </label>
          <label class="settings-toggle-row settings-card-row">
            <span>Clear search when launcher opens</span>
            <span class="modern-switch">
              <input v-model="cfg.ui.clearQueryOnShow" type="checkbox" />
              <span class="modern-switch-track" aria-hidden="true"></span>
            </span>
          </label>
        </div>
      </section>

      <section class="settings-section">
        <h2>Shortcuts</h2>
        <div class="settings-card">
          <button
            class="settings-action-row"
            :class="{ recording: recordingHotkey }"
            type="button"
            @click="$emit('toggle-hotkey-recording')"
            @keydown="$emit('capture-hotkey', $event)"
          >
            <span class="settings-row-label"
              ><span class="settings-symbol"
                ><svg viewBox="0 0 24 24" aria-hidden="true">
                  <path :d="icons.keyboard" /></svg></span
              >Open launcher</span
            >
            <code>{{ recordingHotkey ? "Press keys" : cfg.hotkey.binding }}</code>
          </button>
          <label class="settings-action-row workspace-trigger-row">
            <span class="settings-row-label"
              ><span class="settings-symbol"
                ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.grid" /></svg></span
              >Workspace filter key</span
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
        <h2>Process mode</h2>
        <div class="mode-card-grid">
          <button
            class="process-card"
            :class="{ selected: standardModeSelected }"
            type="button"
            :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'standard'"
            @click="$emit('switch-elevation', 'standard')"
          >
            <span class="process-glyph"
              ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.account" /></svg
            ></span>
            <span class="process-radio" aria-hidden="true"></span>
            <strong>Standard</strong>
            <small>Runs with normal privileges. Safer, but can't reach elevated apps.</small>
          </button>
          <button
            class="process-card admin"
            :class="{ selected: adminModeSelected }"
            type="button"
            :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'admin'"
            @click="$emit('switch-elevation', 'admin')"
          >
            <span class="process-glyph"
              ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.shieldAccount" /></svg
            ></span>
            <span class="process-radio" aria-hidden="true"></span>
            <strong>Administrator</strong>
            <small>Requires UAC. Can launch and reach elevated apps.</small>
          </button>
          <small class="mode-message">{{
            elevationStatus?.message ?? "Checking process mode"
          }}</small>
        </div>
      </section>

      <section class="settings-section">
        <div class="settings-section-title-row">
          <h2>Indexed folders</h2>
          <div class="settings-section-actions">
            <button
              class="settings-link-button"
              type="button"
              :disabled="scanning"
              @click="$emit('scan')"
            >
              {{ scanning ? "Scanning" : "Scan" }}
            </button>
            <button
              class="settings-link-button"
              :class="{ active: cfg.scanner.watch }"
              type="button"
              :aria-pressed="cfg.scanner.watch"
              title="Automatically update the index on file changes"
              @click="cfg.scanner.watch = !cfg.scanner.watch"
            >
              Watch
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
              <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.delete" /></svg>
            </button>
          </div>
          <p v-if="!cfg.scanner.roots.length" class="root-empty-hint">
            Add a folder to index your projects.
          </p>
        </div>
        <button class="settings-add-button" type="button" @click="$emit('browse-root')">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.add" /></svg>Add folder
        </button>
      </section>

      <section v-if="wslPresent" class="settings-section">
        <div class="settings-section-title-row">
          <h2>WSL folders</h2>
        </div>
        <div class="settings-card">
          <div
            v-for="(root, index) in cfg.scanner.wslRoots ?? []"
            :key="(root.distro ?? '') + root.path"
            class="root-row"
          >
            <span class="root-path"
              ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.folder" /></svg
              >{{ root.path
              }}<small v-if="root.distro" class="root-distro">{{ root.distro }}</small></span
            >
            <button type="button" title="Remove folder" @click="$emit('remove-wsl-root', index)">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.delete" /></svg>
            </button>
          </div>
          <p v-if="!cfg.scanner.wslRoots?.length" class="root-empty-hint">
            Pick a folder from \\wsl.localhost; it is stored and shown as a Unix path.
          </p>
        </div>
        <button class="settings-add-button" type="button" @click="$emit('browse-wsl-root')">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.add" /></svg>Add WSL folder
        </button>
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
    </main>

    <nav class="settings-bottom-bar">
      <button class="settings-text-button" type="button" @click="$emit('close')">Cancel</button>
      <button class="settings-save-button" type="button" @click="$emit('save')">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.check" /></svg>Save
      </button>
    </nav>
  </div>
</template>
