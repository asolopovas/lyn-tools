<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    variant?: "text" | "filled" | "link" | "add";
    active?: boolean;
    disabled?: boolean;
  }>(),
  { variant: "text", active: false, disabled: false },
);

const base =
  "inline-flex h-9 items-center justify-center gap-2 rounded-[10px] border border-transparent text-[13px]/[18px] tracking-[0.1px] font-semibold whitespace-nowrap transition-[background,border-color,box-shadow,transform] duration-140 ease-[ease] active:translate-y-px [&_svg]:size-4 [&_svg]:fill-current";

const classes = computed(() => {
  switch (props.variant) {
    case "filled":
      return `${base} bg-(--m3-accent) pr-[18px] pl-3.5 text-(--m3-on-accent) shadow-[0_1px_2px_rgba(0,0,0,.32),inset_0_1px_0_rgba(255,255,255,.16)] hover:bg-[color-mix(in_srgb,var(--m3-accent)_90%,#ffffff)] hover:shadow-[0_4px_12px_rgba(0,0,0,.34),inset_0_1px_0_rgba(255,255,255,.18)] active:shadow-[inset_0_1px_2px_rgba(0,0,0,.28)] disabled:opacity-45 disabled:shadow-none`;
    case "add":
      return `${base} self-end border-(--m3-outline-strong) bg-transparent pr-4 pl-3 text-(--m3-accent) hover:border-(--m3-accent) hover:bg-(--m3-state) active:bg-(--m3-state-strong) disabled:opacity-45`;
    case "link":
      return props.active
        ? `${base} bg-(--m3-accent) px-4 text-(--m3-on-accent) shadow-[0_1px_2px_rgba(0,0,0,.28),inset_0_1px_0_rgba(255,255,255,.14)] active:shadow-[inset_0_1px_2px_rgba(0,0,0,.28)]`
        : `${base} border-(--m3-outline-strong) bg-transparent px-4 text-(--m3-accent) not-disabled:hover:border-(--m3-accent) not-disabled:hover:bg-(--m3-state) active:bg-(--m3-state-strong) disabled:opacity-[0.38]`;
    default:
      return `${base} bg-transparent px-4 text-(--m3-accent) hover:bg-(--m3-state) active:bg-(--m3-state-strong) disabled:opacity-45`;
  }
});
</script>

<template>
  <button :class="classes" type="button" :disabled="disabled">
    <slot />
  </button>
</template>
