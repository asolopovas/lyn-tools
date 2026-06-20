<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string;
    inline?: boolean;
    labelClass?: string;
  }>(),
  { inline: false, labelClass: "" },
);
</script>

<template>
  <label
    class="settings-field relative gap-1.5 px-3.5 py-2.5 text-[12px]/4 tracking-[0.4px] text-(--m3-on-surface-variant) after:absolute after:inset-x-3.5 after:bottom-0 after:h-px after:bg-(--m3-outline) after:content-[''] last:after:hidden"
    :class="inline ? 'grid grid-cols-[1fr_auto] items-center' : 'grid'"
  >
    <span class="text-[13px]/[18px] tracking-[0.1px] text-(--m3-on-surface)" :class="labelClass">
      <slot name="label">{{ label }}</slot>
    </span>
    <slot />
  </label>
</template>
