<script setup lang="ts">
import { consumeEvent } from "../hotkeys";
import { icons } from "../icons";
import { detail, title } from "../projectUtils";
import { actionButtons, isSystemCommand, systemCommandIcon } from "../resultActions";
import type { Project, ProjectAction } from "../types";

const props = defineProps<{
  project: Project;
  selected: boolean;
  icon: string;
}>();

const emit = defineEmits<{
  launch: [project: Project];
  action: [project: Project, action: ProjectAction];
  select: [];
}>();

function launchResult(event: PointerEvent): void {
  consumeEvent(event, false);
  emit("launch", props.project);
}
</script>

<template>
  <li :class="{ selected }" @pointermove="emit('select')" @pointerdown="launchResult">
    <div class="project-icon" aria-hidden="true">
      <svg v-if="isSystemCommand(project)" viewBox="0 0 24 24">
        <path :d="icons[systemCommandIcon(project)]" />
      </svg>
      <img v-else-if="icon" :src="icon" alt="" />
      <svg v-else viewBox="0 0 24 24"><path :d="icons.folder" /></svg>
    </div>
    <div class="project-copy">
      <strong>{{ title(project) }}</strong>
      <small>{{ detail(project) }}</small>
    </div>
    <div class="result-actions" aria-label="Result actions" @pointerdown="consumeEvent">
      <button
        v-for="button in actionButtons(project)"
        :key="button.action"
        class="result-action-button"
        type="button"
        :title="button.title"
        @click="emit('action', project, button.action)"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons[button.icon]" /></svg>
      </button>
    </div>
  </li>
</template>
