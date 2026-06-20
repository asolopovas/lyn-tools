<script setup lang="ts">
import UiToggle from "./UiToggle.vue";

const model = defineModel<boolean>({ required: true });

defineProps<{
  label: string;
  disabled?: boolean;
}>();
</script>

<template>
  <label
    class="flex min-h-5 cursor-pointer items-center justify-between gap-3 px-3.5 py-2.5 text-[13px]/[18px] tracking-[0.1px] text-(--m3-on-surface)"
  >
    <span>{{ label }}</span>
    <UiToggle v-model="model" :disabled="disabled" />
  </label>
</template>
