<script setup lang="ts">
const model = defineModel<boolean>({ required: true });

defineProps<{
  disabled?: boolean;
}>();
</script>

<template>
  <span class="modern-switch relative h-5 w-9 flex-none">
    <input
      class="absolute inset-0 m-0 cursor-pointer opacity-0"
      v-model="model"
      type="checkbox"
      :disabled="disabled"
    />
    <span class="modern-switch-track" aria-hidden="true"></span>
  </span>
</template>

<style scoped>
.modern-switch-track {
  position: absolute;
  inset: 0;
  border: 2px solid var(--m3-outline-strong);
  border-radius: 999px;
  background: var(--m3-surface-container-highest);
  transition:
    background 160ms ease,
    border-color 160ms ease;
}
.modern-switch-track::after {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--m3-on-surface-variant);
  content: "";
  transition:
    transform 160ms cubic-bezier(0.2, 0, 0, 1),
    width 160ms ease,
    height 160ms ease,
    top 160ms ease,
    left 160ms ease,
    background 160ms ease;
}
.modern-switch input:checked + .modern-switch-track {
  background: var(--m3-accent);
  border-color: var(--m3-accent);
}
.modern-switch input:checked + .modern-switch-track::after {
  top: 0;
  left: 0;
  width: 16px;
  height: 16px;
  transform: translateX(16px);
  background: var(--m3-on-accent);
}
.modern-switch input:disabled + .modern-switch-track {
  opacity: 0.38;
}
</style>
