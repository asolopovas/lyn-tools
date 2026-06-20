<script setup lang="ts">
import { computed } from "vue";
import { icons } from "../icons";
import { themeByKey } from "../themes";
import type { ElevationMode, ElevationStatus, LynConfig } from "../types";
import UiButton from "./ui/UiButton.vue";
import UiCard from "./ui/UiCard.vue";
import UiCardRow from "./ui/UiCardRow.vue";
import UiField from "./ui/UiField.vue";
import UiIconButton from "./ui/UiIconButton.vue";
import UiSection from "./ui/UiSection.vue";
import UiSelect from "./ui/UiSelect.vue";
import UiSlider from "./ui/UiSlider.vue";
import UiSwatch from "./ui/UiSwatch.vue";
import UiToggle from "./ui/UiToggle.vue";

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
              <UiSelect v-model="cfg.ui.theme">
                <option v-for="key in themeKeys" :key="key" :value="key">
                  {{ themeByKey(key).name }}
                </option>
              </UiSelect>
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
            <label
              class="flex min-h-5 cursor-pointer items-center justify-between gap-3 px-3.5 py-2.5 text-[13px]/[18px] tracking-[0.1px] text-(--m3-on-surface)"
            >
              <span>Clear search on open</span>
              <UiToggle v-model="cfg.ui.clearQueryOnShow" />
            </label>
          </UiCard>
        </UiSection>

        <UiSection title="Shortcuts">
          <UiCard>
            <button
              class="settings-action-row relative flex min-h-11 items-center justify-between gap-3 rounded-none border-0 bg-transparent px-3.5 text-left transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state) after:absolute after:inset-x-3.5 after:bottom-0 after:h-px after:bg-(--m3-outline) after:content-[''] last:after:hidden"
              :class="{
                'text-(--m3-accent)': recordingHotkey,
                'text-(--m3-on-surface)': !recordingHotkey,
              }"
              type="button"
              @click="$emit('toggle-hotkey-recording')"
              @keydown="$emit('capture-hotkey', $event)"
            >
              <span class="flex min-w-0 items-center gap-3 text-[13px]/[18px] tracking-[0.1px]"
                ><span
                  class="inline-flex text-(--m3-on-surface-variant) [&_svg]:size-5 [&_svg]:fill-current"
                  ><svg viewBox="0 0 24 24" aria-hidden="true">
                    <path :d="icons.keyboard" /></svg></span
                >Open launcher</span
              >
              <code
                class="flex-none rounded-md border-0 bg-(--m3-surface-container-highest) px-2.5 py-1.25 font-mono text-[12px]/4 text-(--m3-on-surface)"
                >{{ recordingHotkey ? "Press keys" : cfg.hotkey.binding }}</code
              >
            </button>
            <label
              class="settings-action-row relative flex min-h-11 items-center justify-between gap-3 rounded-none border-0 bg-transparent px-3.5 text-left text-(--m3-on-surface) transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state) after:absolute after:inset-x-3.5 after:bottom-0 after:h-px after:bg-(--m3-outline) after:content-[''] last:after:hidden"
            >
              <span class="flex min-w-0 items-center gap-3 text-[13px]/[18px] tracking-[0.1px]"
                ><span
                  class="inline-flex text-(--m3-on-surface-variant) [&_svg]:size-5 [&_svg]:fill-current"
                  ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.grid" /></svg></span
                >Workspace key</span
              >
              <input
                class="h-9 w-14 rounded-lg border border-(--m3-outline-strong) bg-(--m3-surface-container-high) px-3 text-center font-mono text-[12px]/4 text-(--m3-on-surface) focus:border-(--m3-accent) focus:shadow-[0_0_0_1px_var(--m3-accent)]"
                v-model="cfg.ui.workspaceQueryShortcut"
                maxlength="1"
                spellcheck="false"
                @input="$emit('normalize-workspace-shortcut')"
              />
            </label>
          </UiCard>
        </UiSection>

        <section v-if="platform === 'windows'" class="flex flex-col gap-1.5">
          <h2
            class="m-0 ml-1 text-[11px]/4 font-bold tracking-[0.55px] text-(--m3-accent) uppercase"
          >
            Process mode
          </h2>
          <div class="grid grid-cols-2 gap-3">
            <button
              class="process-card relative grid min-h-23 gap-1 rounded-[14px] border bg-(--m3-surface-container) py-3 pr-10 pl-3.5 text-left text-(--m3-on-surface) transition-[background,border-color] duration-140 ease-[ease] hover:bg-(--m3-surface-container-high) disabled:cursor-default"
              :class="
                standardModeSelected
                  ? 'border-transparent bg-(--m3-accent-container)! selected'
                  : 'border-(--m3-outline)'
              "
              type="button"
              :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'standard'"
              @click="$emit('switch-elevation', 'standard')"
            >
              <span
                v-if="standardModeSelected"
                class="absolute top-0 right-0 grid size-6 place-items-center rounded-bl-xl bg-(--m3-accent) [&_svg]:size-3.5 [&_svg]:fill-(--m3-on-accent)"
                aria-hidden="true"
                ><svg viewBox="0 0 24 24"><path :d="icons.check" /></svg
              ></span>
              <span
                class="inline-flex [&_svg]:size-6 [&_svg]:fill-current"
                :class="
                  standardModeSelected ? 'text-(--m3-accent)' : 'text-(--m3-on-surface-variant)'
                "
                ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.account" /></svg
              ></span>
              <strong class="text-(--m3-on-surface) text-sm/[18px] font-semibold">Standard</strong>
              <small class="text-[12px]/4 whitespace-normal text-(--m3-on-surface-variant)"
                >Runs with normal privileges. Safer, but can't reach elevated apps.</small
              >
            </button>
            <button
              class="process-card admin relative grid min-h-23 gap-1 rounded-[14px] border bg-(--m3-surface-container) py-3 pr-10 pl-3.5 text-left text-(--m3-on-surface) transition-[background,border-color] duration-140 ease-[ease] hover:bg-(--m3-surface-container-high) disabled:cursor-default"
              :class="
                adminModeSelected
                  ? 'border-transparent bg-(--m3-accent-container)! selected'
                  : 'border-(--m3-outline)'
              "
              type="button"
              :disabled="!elevationStatus?.canSwitch || elevationStatus?.mode === 'admin'"
              @click="$emit('switch-elevation', 'admin')"
            >
              <span
                v-if="adminModeSelected"
                class="absolute top-0 right-0 grid size-6 place-items-center rounded-bl-xl bg-(--m3-accent) [&_svg]:size-3.5 [&_svg]:fill-(--m3-on-accent)"
                aria-hidden="true"
                ><svg viewBox="0 0 24 24"><path :d="icons.check" /></svg
              ></span>
              <span
                class="inline-flex [&_svg]:size-6 [&_svg]:fill-current"
                :class="adminModeSelected ? 'text-(--m3-accent)' : 'text-(--m3-error)'"
                ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.shieldAccount" /></svg
              ></span>
              <strong class="text-(--m3-on-surface) text-sm/[18px] font-semibold"
                >Administrator</strong
              >
              <small class="text-[12px]/4 whitespace-normal text-(--m3-on-surface-variant)"
                >Requires UAC. Can launch and reach elevated apps.</small
              >
            </button>
            <small class="px-1 text-[12px]/4 whitespace-normal text-(--m3-on-surface-variant)">{{
              elevationStatus?.message ?? "Checking process mode"
            }}</small>
          </div>
        </section>
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
            <UiCardRow
              v-for="(root, index) in cfg.scanner.roots"
              :key="root"
              class="root-row flex min-h-11 items-center justify-between gap-2.5 py-0! pr-1.5! pl-3.5! transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state)"
            >
              <span
                class="flex min-w-0 items-center gap-3 overflow-hidden text-ellipsis whitespace-nowrap font-mono text-[12px]/4 text-(--m3-on-surface) [&_svg]:size-5 [&_svg]:flex-none [&_svg]:fill-none [&_svg]:stroke-(--m3-on-surface-variant) [&_svg]:stroke-[1.8]"
                ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.folder" /></svg
                >{{ root }}</span
              >
              <UiIconButton
                variant="danger"
                title="Remove folder"
                @click="$emit('remove-root', index)"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.delete" /></svg>
              </UiIconButton>
            </UiCardRow>
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
            <UiCardRow
              v-for="(root, index) in cfg.scanner.wslRoots ?? []"
              :key="(root.distro ?? '') + root.path"
              class="root-row flex min-h-11 items-center justify-between gap-2.5 py-0! pr-1.5! pl-3.5! transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state)"
            >
              <span
                class="flex min-w-0 items-center gap-3 overflow-hidden text-ellipsis whitespace-nowrap font-mono text-[12px]/4 text-(--m3-on-surface) [&_svg]:size-5 [&_svg]:flex-none [&_svg]:fill-none [&_svg]:stroke-(--m3-on-surface-variant) [&_svg]:stroke-[1.8]"
                ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.folder" /></svg
                >{{ root.path
                }}<small
                  v-if="root.distro"
                  class="ml-2 flex-none rounded-full bg-(--m3-surface-container-highest) px-2 py-0.5 text-[10px]/[14px] text-(--m3-on-surface-variant)"
                  >{{ root.distro }}</small
                ></span
              >
              <UiIconButton
                variant="danger"
                title="Remove folder"
                @click="$emit('remove-wsl-root', index)"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.delete" /></svg>
              </UiIconButton>
            </UiCardRow>
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
            <label
              class="flex min-h-5 cursor-pointer items-center justify-between gap-3 px-3.5 py-2.5 text-[13px]/[18px] tracking-[0.1px] text-(--m3-on-surface)"
            >
              <span>Start Lyn with the system</span>
              <UiToggle v-model="cfg.startup.enabled" />
            </label>
            <label
              class="flex min-h-5 cursor-pointer items-center justify-between gap-3 px-3.5 py-2.5 text-[13px]/[18px] tracking-[0.1px] text-(--m3-on-surface)"
            >
              <span>Run in background</span>
              <UiToggle v-model="cfg.startup.startHidden" :disabled="!cfg.startup.enabled" />
            </label>
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
