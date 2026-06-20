<script setup lang="ts">
const model = defineModel<number>({ required: true });

defineProps<{
  min: number | string;
  max: number | string;
  step: number | string;
}>();
</script>

<template>
  <input
    class="range-input h-5 w-full appearance-none border-0 bg-transparent p-0"
    v-model.number="model"
    type="range"
    :min="min"
    :max="max"
    :step="step"
  />
</template>

<style scoped>
.range-input::-webkit-slider-runnable-track {
  height: 4px;
  border-radius: 999px;
  background: var(--m3-surface-container-highest);
}
.range-input::-webkit-slider-thumb {
  appearance: none;
  width: 14px;
  height: 14px;
  margin-top: -5px;
  border: 2px solid var(--m3-surface-container);
  border-radius: 50%;
  background: var(--m3-accent);
}
</style>
