<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    variant?: "default" | "danger";
  }>(),
  { variant: "default" },
);

const classes = computed(() =>
  props.variant === "danger"
    ? "size-8 rounded-full border-0 bg-transparent p-0 text-(--m3-on-surface-variant) transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state-strong) hover:text-(--m3-error) [&_svg]:size-[18px] [&_svg]:fill-current"
    : "grid size-8 place-items-center rounded-full border-0 bg-transparent p-0 text-(--m3-on-surface-variant) transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state) hover:text-(--m3-on-surface) [&_svg]:size-5 [&_svg]:fill-current",
);
</script>

<template>
  <button :class="classes" type="button">
    <slot />
  </button>
</template>
