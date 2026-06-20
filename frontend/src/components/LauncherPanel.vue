<script setup lang="ts">
import { consumeEvent, launcherActionForKey } from "../hotkeys";
import { icons } from "../icons";
import type { Project, ProjectAction } from "../types";
import LauncherResult from "./LauncherResult.vue";

const query = defineModel<string>("query", { required: true });

defineProps<{
  matches: Project[];
  selectedIndex: number;
  loading: boolean;
  statusLine: string;
  scanning: boolean;
  projectIcons: Record<string, string>;
  platform: string;
}>();

const emit = defineEmits<{
  "launch-selected": [action: ProjectAction];
  "open-settings": [];
  launch: [project: Project];
  action: [project: Project, action: ProjectAction];
  select: [index: number];
  scan: [];
}>();

function onQuerySubmit(event: SubmitEvent): void {
  consumeEvent(event, false);
  emit("launch-selected", "code");
}

function emitResultAction(project: Project, action: ProjectAction): void {
  emit("action", project, action);
}

function onQueryKeydown(event: KeyboardEvent): void {
  const action = launcherActionForKey(event);
  if (!action) {
    return;
  }
  consumeEvent(event);
  emit("launch-selected", action);
}
</script>

<template>
  <form
    class="grid h-16 grid-cols-[4rem_1fr_3.5rem] items-center border-b border-line"
    @submit="onQuerySubmit"
  >
    <svg class="size-5 justify-self-center fill-fg" viewBox="0 0 28 28" aria-hidden="true">
      <path :d="icons.search" />
    </svg>
    <input
      v-model="query"
      data-lyn-query="true"
      autofocus
      spellcheck="false"
      placeholder="Start typing..."
      class="size-full border-0 bg-transparent text-base leading-normal text-fg caret-fg outline-none placeholder:text-fg"
      @keydown="onQueryKeydown"
    />
    <button
      class="grid size-9 place-items-center justify-self-center rounded border border-transparent bg-transparent p-0 text-fg hover:bg-selected"
      type="button"
      title="Settings"
      @click="emit('open-settings')"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true" class="size-5 fill-none stroke-current stroke-1">
        <path :d="icons.settings" />
      </svg>
    </button>
  </form>
  <ul
    v-if="matches.length"
    data-lyn-results="true"
    class="lyn-scroll m-0 box-border h-[calc(100%-4rem)] list-none overflow-x-hidden overflow-y-auto p-0"
  >
    <LauncherResult
      v-for="(project, index) in matches"
      :key="project.path"
      :project="project"
      :selected="index === selectedIndex"
      :icon="projectIcons[project.path] ?? ''"
      :platform="platform"
      @select="emit('select', index)"
      @launch="emit('launch', $event)"
      @action="emitResultAction"
    />
  </ul>
  <div v-else-if="loading" class="grid content-start gap-1.5 p-5">
    <strong class="text-base font-normal leading-5 text-fg">Loading projects</strong>
    <small class="text-xs leading-4 text-muted">{{ statusLine }}</small>
  </div>
  <div v-else class="grid content-start gap-1.5 p-5">
    <strong class="text-base font-normal leading-5 text-fg">No projects found</strong>
    <small class="text-xs leading-4 text-muted">{{ statusLine }}</small>
    <button
      type="button"
      :disabled="scanning"
      class="cursor-pointer justify-self-start rounded border border-line/55 bg-panel-alt px-2.5 py-1 text-xs font-semibold text-fg hover:border-accent disabled:opacity-55"
      @click="emit('scan')"
    >
      {{ scanning ? "Scanning" : "Scan" }}
    </button>
  </div>
</template>

<style scoped>
.lyn-scroll {
  scrollbar-color: color-mix(in srgb, var(--lyn-text) 18%, transparent) transparent;
  scrollbar-width: thin;
}
.lyn-scroll::-webkit-scrollbar {
  width: 8px;
}
.lyn-scroll::-webkit-scrollbar-track {
  background: transparent;
}
.lyn-scroll::-webkit-scrollbar-thumb {
  border: solid transparent;
  border-width: 8px 0;
  border-radius: 999px;
  background: color-mix(in srgb, var(--lyn-text) 18%, transparent);
  background-clip: content-box;
}
.lyn-scroll::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--lyn-text) 28%, transparent);
  background-clip: content-box;
}
</style>
