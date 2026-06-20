<script setup lang="ts">
const model = defineModel<string>({ required: true });
</script>

<template>
  <select
    class="settings-select h-9 appearance-none rounded-lg border border-(--m3-outline-strong) bg-(--m3-surface-container-high) py-0 pr-9 pl-3 text-[13px]/[18px] tracking-[0.1px] text-(--m3-on-surface) focus:border-(--m3-accent) focus:shadow-[0_0_0_1px_var(--m3-accent)]"
    v-model="model"
  >
    <slot />
  </select>
</template>

<style scoped>
.settings-select {
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='%23a8a8a8' d='M7.4 9.6 12 14.2l4.6-4.6 1.4 1.4-6 6-6-6z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  background-size: 18px 18px;
}
</style>
