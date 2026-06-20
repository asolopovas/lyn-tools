<script setup lang="ts">
import { computed } from "vue";
import { icons } from "../icons";
import { themeByKey } from "../themes";
import type { ElevationMode, ElevationStatus, LynConfig } from "../types";
import UiButton from "./ui/UiButton.vue";
import UiCard from "./ui/UiCard.vue";
import UiField from "./ui/UiField.vue";
import UiIconButton from "./ui/UiIconButton.vue";
import UiNavRow from "./ui/UiNavRow.vue";
import UiOptionCard from "./ui/UiOptionCard.vue";
import UiPathRow from "./ui/UiPathRow.vue";
import UiSection from "./ui/UiSection.vue";
import UiSelect from "./ui/UiSelect.vue";
import UiSlider from "./ui/UiSlider.vue";
import UiSwatch from "./ui/UiSwatch.vue";
import UiToggleRow from "./ui/UiToggleRow.vue";

const cfg = defineModel<LynConfig>("cfg", { required: true });

const props = defineProps<{
  themeKeys: string[];
  scanning: boolean;
  platform: string;
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

const themeOptions = computed(() =>
  props.themeKeys.map((key) => ({ value: key, label: themeByKey(key).name })),
);
const opacityPercent = computed(() => `${Math.round(cfg.value.ui.backgroundOpacity * 100)}%`);
const standardModeSelected = computed(() => props.elevationStatus?.mode !== "admin");
const adminModeSelected = computed(() => props.elevationStatus?.mode === "admin");
</script>

<template>
  <div
    class="settings-panel grid h-full max-h-full w-full grid-rows-[48px_1fr_auto] overflow-hidden bg-(--m3-surface) text-(--m3-on-surface) [font-optical-sizing:auto]"
  >
    <header
      class="settings-topbar flex items-center justify-between bg-(--m3-surface) pr-2 pl-4 [--wails-draggable:drag]"
    >
      <div class="flex min-w-0 items-center gap-3">
        <svg class="size-5 fill-(--m3-accent)" viewBox="0 0 24 24" aria-hidden="true">
          <path :d="icons.settings" />
        </svg>
        <h1
          class="m-0 overflow-hidden text-ellipsis whitespace-nowrap text-(--m3-on-surface) text-lg/6 font-medium"
        >
          Settings
        </h1>
      </div>
      <UiIconButton title="Close settings" @click="$emit('close')">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.close" /></svg>
      </UiIconButton>
    </header>

    <main
      class="settings-content grid min-h-0 grid-cols-[repeat(auto-fit,minmax(220px,1fr))] items-start gap-4 overflow-x-hidden overflow-y-auto px-4 pt-2.5 pb-3.5 [scrollbar-color:var(--m3-outline-strong)_transparent] scrollbar-thin"
    >
      <div class="flex min-w-0 flex-col gap-4">
        <UiSection title="Appearance">
          <UiCard>
            <UiField label="Theme">
              <UiSelect v-model="cfg.ui.theme" :options="themeOptions" />
            </UiField>
            <UiField label="Quick pick" inline>
              <div class="flex flex-wrap justify-end gap-2">
                <UiSwatch
                  v-for="key in themeKeys"
                  :key="key"
                  :accent="themeByKey(key).accent"
                  :bg="themeByKey(key).background"
                  :active="cfg.ui.theme === key"
                  :label="themeByKey(key).name"
                  @click="cfg.ui.theme = key"
                />
              </div>
            </UiField>
            <UiField label="Opacity" label-class="flex items-center justify-between gap-2.5">
              <template #label
                ><span>Opacity</span
                ><code class="font-mono text-[12px]/4">{{ opacityPercent }}</code></template
              >
              <UiSlider v-model="cfg.ui.backgroundOpacity" min="0.55" max="1" step="0.01" />
            </UiField>
            <UiToggleRow label="Clear search on open" v-model="cfg.ui.clearQueryOnShow" />
          </UiCard>
        </UiSection>

        <UiSection title="Shortcuts">
          <UiCard>
            <UiNavRow
              as="button"
              type="button"
              :icon="icons.keyboard"
              label="Open launcher"
              :active="recordingHotkey"
              @click="$emit('toggle-hotkey-recording')"
              @keydown="$emit('capture-hotkey', $event)"
            >
              <code
                class="flex-none rounded-md border-0 bg-(--m3-surface-container-highest) px-2.5 py-1.25 font-mono text-[12px]/4 text-(--m3-on-surface)"
                >{{ recordingHotkey ? "Press keys" : cfg.hotkey.binding }}</code
              >
            </UiNavRow>
            <UiNavRow :icon="icons.grid" label="Workspace key">
              <input
                class="h-9 w-14 rounded-lg border border-(--m3-outline-strong) bg-(--m3-surface-container-high) px-3 text-center font-mono text-[12px]/4 text-(--m3-on-surface) focus:border-(--m3-accent) focus:shadow-[0_0_0_1px_var(--m3-accent)]"
                v-model="cfg.ui.workspaceQueryShortcut"
                maxlength="1"
                spellcheck="false"
                @input="$emit('normalize-workspace-shortcut')"
              />
            </UiNavRow>
          </UiCard>
        </UiSection>

        <UiSection v-if="platform === 'windows'" title="Process mode">
          <div class="grid grid-cols-2 gap-3">
            <UiOptionCard
              :icon="icons.account"
              title="Standard"
              description="Runs with normal privileges. Safer, but can't reach elevated apps."
              :selected="standardModeSelected"
              :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'standard'"
              @select="$emit('switch-elevation', 'standard')"
            />
            <UiOptionCard
              :icon="icons.shieldAccount"
              title="Administrator"
              description="Requires UAC. Can launch and reach elevated apps."
              tone="error"
              :selected="adminModeSelected"
              :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'admin'"
              @select="$emit('switch-elevation', 'admin')"
            />
            <small class="px-1 text-[12px]/4 whitespace-normal text-(--m3-on-surface-variant)">{{
              elevationStatus?.message ?? "Checking process mode"
            }}</small>
          </div>
        </UiSection>
      </div>

      <div class="flex min-w-0 flex-col gap-4">
        <UiSection title="Folders">
          <template #actions>
            <div class="flex items-center gap-2">
              <UiButton variant="link" :disabled="scanning" @click="$emit('scan')">
                {{ scanning ? "Scanning" : "Scan" }}
              </UiButton>
              <UiButton
                variant="link"
                :active="cfg.scanner.watch"
                :aria-pressed="cfg.scanner.watch"
                title="Automatically update the index on file changes"
                @click="cfg.scanner.watch = !cfg.scanner.watch"
              >
                Watch
              </UiButton>
            </div>
          </template>
          <UiCard>
            <UiPathRow
              v-for="(root, index) in cfg.scanner.roots"
              :key="root"
              :path="root"
              @remove="$emit('remove-root', index)"
            />
            <p
              v-if="!cfg.scanner.roots.length"
              class="m-0 p-3.5 text-[12px]/4 tracking-[0.4px] text-(--m3-on-surface-variant)"
            >
              Add a folder to index your projects.
            </p>
          </UiCard>
          <UiButton variant="add" @click="$emit('browse-root')">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.add" /></svg>Add folder
          </UiButton>
        </UiSection>

        <UiSection v-if="wslPresent" title="WSL folders">
          <UiCard>
            <UiPathRow
              v-for="(root, index) in cfg.scanner.wslRoots ?? []"
              :key="(root.distro ?? '') + root.path"
              :path="root.path"
              :badge="root.distro"
              @remove="$emit('remove-wsl-root', index)"
            />
            <p
              v-if="!cfg.scanner.wslRoots?.length"
              class="m-0 p-3.5 text-[12px]/4 tracking-[0.4px] text-(--m3-on-surface-variant)"
            >
              Pick a folder from \\wsl.localhost; it is stored and shown as a Unix path.
            </p>
          </UiCard>
          <UiButton variant="add" @click="$emit('browse-wsl-root')">
            <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.add" /></svg>Add folder
          </UiButton>
        </UiSection>

        <UiSection title="Startup">
          <UiCard>
            <UiToggleRow label="Start Lyn with the system" v-model="cfg.startup.enabled" />
            <UiToggleRow
              label="Run in background"
              v-model="cfg.startup.startHidden"
              :disabled="!cfg.startup.enabled"
            />
          </UiCard>
        </UiSection>
      </div>
    </main>

    <nav
      class="flex items-center justify-end gap-2 border-t border-(--m3-outline) bg-(--m3-surface) px-4 py-2.5"
    >
      <UiButton variant="text" @click="$emit('close')"> Cancel </UiButton>
      <UiButton variant="filled" class="settings-save-button" @click="$emit('save')">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.check" /></svg>Save
      </UiButton>
    </nav>
  </div>
</template>

<style scoped>
.settings-content::-webkit-scrollbar {
  width: 10px;
}
.settings-content::-webkit-scrollbar-thumb {
  border: 3px solid var(--lyn-bg);
  border-radius: 10px;
  background: var(--lyn-muted);
}
</style>
