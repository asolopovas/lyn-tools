<script setup lang="ts">
import { computed } from "vue";
import { icons } from "../../icons";

const props = withDefaults(
  defineProps<{
    icon: string;
    title: string;
    description: string;
    selected: boolean;
    disabled?: boolean;
    tone?: "default" | "error";
  }>(),
  { disabled: false, tone: "default" },
);

defineEmits<{
  select: [];
}>();

const iconClass = computed(() =>
  props.selected
    ? "text-(--m3-accent)"
    : props.tone === "error"
      ? "text-(--m3-error)"
      : "text-(--m3-on-surface-variant)",
);
</script>

<template>
  <button
    class="relative grid min-h-23 gap-1 rounded-[14px] border bg-(--m3-surface-container) py-3 pr-10 pl-3.5 text-left text-(--m3-on-surface) transition-[background,border-color] duration-140 ease-[ease] hover:bg-(--m3-surface-container-high) disabled:cursor-default"
    :class="selected ? 'border-transparent bg-(--m3-accent-container)!' : 'border-(--m3-outline)'"
    type="button"
    :disabled="disabled"
    @click="$emit('select')"
  >
    <span
      v-if="selected"
      class="absolute top-0 right-0 grid size-6 place-items-center rounded-bl-xl bg-(--m3-accent) [&_svg]:size-3.5 [&_svg]:fill-(--m3-on-accent)"
      aria-hidden="true"
      ><svg viewBox="0 0 24 24"><path :d="icons.check" /></svg
    ></span>
    <span class="inline-flex [&_svg]:size-6 [&_svg]:fill-current" :class="iconClass"
      ><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icon" /></svg
    ></span>
    <strong class="text-(--m3-on-surface) text-sm/[18px] font-semibold">{{ title }}</strong>
    <small class="text-[12px]/4 whitespace-normal text-(--m3-on-surface-variant)">{{
      description
    }}</small>
  </button>
</template>
