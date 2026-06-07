<script setup lang="ts">
import { consumeEvent } from "../hotkeys";
import { icons } from "../icons";
import { detail, title } from "../projectUtils";
import { actionButtons, isSystemCommand, systemCommandGlyph } from "../resultActions";
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
      <span v-if="isSystemCommand(project)" class="system-command-icon">{{
        systemCommandGlyph(project)
      }}</span>
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
        <span aria-hidden="true">{{ button.glyph }}</span>
      </button>
    </div>
  </li>
</template>
