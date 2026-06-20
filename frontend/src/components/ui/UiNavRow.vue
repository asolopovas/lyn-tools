<script setup lang="ts">
withDefaults(
  defineProps<{
    icon: string;
    label: string;
    as?: "button" | "label";
    active?: boolean;
  }>(),
  { as: "label", active: false },
);
</script>

<template>
  <component
    :is="as"
    class="relative flex min-h-11 items-center justify-between gap-3 rounded-none border-0 bg-transparent px-3.5 text-left transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state) after:absolute after:inset-x-3.5 after:bottom-0 after:h-px after:bg-(--m3-outline) after:content-[''] last:after:hidden"
    :class="active ? 'text-(--m3-accent)' : 'text-(--m3-on-surface)'"
  >
    <span class="flex min-w-0 items-center gap-3 text-[13px]/[18px] tracking-[0.1px]">
      <span class="inline-flex text-(--m3-on-surface-variant) [&_svg]:size-5 [&_svg]:fill-current">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path :d="icon" /></svg>
      </span>
      {{ label }}
    </span>
    <slot />
  </component>
</template>
