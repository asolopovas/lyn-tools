<script setup lang="ts">
import { icons } from "../../icons";
import UiCardRow from "./UiCardRow.vue";
import UiIconButton from "./UiIconButton.vue";

defineProps<{
  path: string;
  badge?: string | undefined;
}>();

defineEmits<{
  remove: [];
}>();
</script>

<template>
  <UiCardRow
    class="flex min-h-11 items-center justify-between gap-2.5 py-0! pr-1.5! pl-3.5! transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state)"
  >
    <span
      class="flex min-w-0 items-center gap-3 overflow-hidden text-ellipsis whitespace-nowrap font-mono text-[12px]/4 text-(--m3-on-surface) [&_svg]:size-5 [&_svg]:flex-none [&_svg]:fill-none [&_svg]:stroke-(--m3-on-surface-variant) [&_svg]:stroke-[1.8]"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.folder" /></svg>
      {{ path }}
      <small
        v-if="badge"
        class="ml-2 flex-none rounded-full bg-(--m3-surface-container-highest) px-2 py-0.5 text-[10px]/[14px] text-(--m3-on-surface-variant)"
        >{{ badge }}</small
      >
    </span>
    <UiIconButton variant="danger" title="Remove folder" @click="$emit('remove')">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.delete" /></svg>
    </UiIconButton>
  </UiCardRow>
</template>
