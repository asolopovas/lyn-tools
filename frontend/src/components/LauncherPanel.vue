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
  <form class="query-row" @submit="onQuerySubmit">
    <svg class="search-icon" viewBox="0 0 28 28" aria-hidden="true">
      <path :d="icons.search" />
    </svg>
    <input
      v-model="query"
      data-lyn-query="true"
      autofocus
      spellcheck="false"
      placeholder="Start typing..."
      @keydown="onQueryKeydown"
    />
    <button
      class="query-action settings-action"
      type="button"
      title="Settings"
      @click="emit('open-settings')"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.settings" /></svg>
    </button>
  </form>
  <ul v-if="matches.length" data-lyn-results="true">
    <LauncherResult
      v-for="(project, index) in matches"
      :key="project.path"
      :project="project"
      :selected="index === selectedIndex"
      :icon="projectIcons[project.path] ?? ''"
      @select="emit('select', index)"
      @launch="emit('launch', $event)"
      @action="emitResultAction"
    />
  </ul>
  <div v-else-if="loading" class="empty">
    <strong>Loading projects</strong>
    <small>{{ statusLine }}</small>
  </div>
  <div v-else class="empty">
    <strong>No projects found</strong>
    <small>{{ statusLine }}</small>
    <button type="button" :disabled="scanning" @click="emit('scan')">
      {{ scanning ? "Scanning" : "Scan" }}
    </button>
  </div>
</template>
