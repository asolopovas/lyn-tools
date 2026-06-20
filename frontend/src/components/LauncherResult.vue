<script setup lang="ts">
import { computed } from "vue";
import { consumeEvent } from "../hotkeys";
import { icons } from "../icons";
import { detail, title } from "../projectUtils";
import { actionButtons, isSystemCommand, systemCommandIcon } from "../resultActions";
import type { Project, ProjectAction } from "../types";

const props = defineProps<{
  project: Project;
  selected: boolean;
  icon: string;
  platform: string;
}>();

const detailText = computed(() => detail(props.project));

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
  <li
    class="relative box-border grid min-h-13 w-full grid-cols-[4rem_minmax(0,1fr)_auto] cursor-pointer items-center pr-3 text-fg"
    :class="[
      { selected },
      selected &&
        `before:pointer-events-none before:absolute before:inset-0 before:z-0 before:content-[''] before:bg-[color-mix(in_srgb,var(--lyn-selected)_72%,transparent)]`,
    ]"
    @pointermove="emit('select')"
    @pointerdown="launchResult"
  >
    <div
      class="relative z-10 grid place-items-center"
      :class="selected ? 'text-selected-fg' : 'text-accent'"
      aria-hidden="true"
    >
      <svg v-if="isSystemCommand(project)" viewBox="0 0 24 24" class="size-6 fill-current">
        <path :d="icons[systemCommandIcon(project)]" />
      </svg>
      <img v-else-if="icon" :src="icon" alt="" class="size-6 object-contain" />
      <svg v-else viewBox="0 0 24 24" class="size-6 fill-current"><path :d="icons.folder" /></svg>
    </div>
    <div class="relative z-10 grid min-w-0">
      <strong
        class="overflow-hidden text-ellipsis whitespace-nowrap text-base font-normal leading-5"
        :class="selected ? 'text-selected-fg' : 'text-fg'"
        >{{ title(project) }}</strong
      >
      <small
        v-if="detailText"
        class="overflow-hidden text-ellipsis whitespace-nowrap text-xs leading-4"
        :class="selected ? 'text-selected-fg' : 'text-muted'"
        >{{ detailText }}</small
      >
    </div>
    <div
      class="relative z-10 flex items-center justify-end gap-0.5 pl-2.5"
      aria-label="Result actions"
      @pointerdown="consumeEvent"
    >
      <button
        v-for="button in actionButtons(project, platform)"
        :key="button.action"
        class="grid size-8 place-items-center border-transparent bg-transparent p-0 hover:bg-selected hover:text-selected-fg"
        :class="selected ? 'text-selected-fg' : 'text-muted'"
        type="button"
        :title="button.title"
        @click="emit('action', project, button.action)"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true" class="size-5 fill-current">
          <path :d="icons[button.icon]" />
        </svg>
      </button>
    </div>
  </li>
</template>
