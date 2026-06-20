<script setup lang="ts">
import { icons } from "../../icons";

defineProps<{
  accent: string;
  bg: string;
  active: boolean;
  label: string;
}>();
</script>

<template>
  <button
    class="theme-swatch grid size-[22px] cursor-pointer place-items-center rounded-full border border-(--m3-outline-strong) bg-(--swatch-accent) p-0 transition-[transform,box-shadow] duration-140 ease-[ease] hover:scale-[1.12] [&_svg]:size-3.5 [&_svg]:fill-(--swatch-bg) [&_svg]:opacity-0"
    :class="{ active }"
    type="button"
    :style="{ '--swatch-accent': accent, '--swatch-bg': bg }"
    :title="label"
    :aria-label="label"
    :aria-pressed="active"
  >
    <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icons.check" /></svg>
  </button>
</template>

<style scoped>
.theme-swatch.active {
  border-color: var(--swatch-bg);
  box-shadow:
    0 0 0 2px var(--m3-surface-container),
    0 0 0 4px var(--m3-accent);
}
.theme-swatch.active svg {
  opacity: 1;
}
</style>
