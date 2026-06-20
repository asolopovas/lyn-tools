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

const classes = computed(() => {
  switch (props.variant) {
    case "filled":
      return "inline-flex h-[34px] items-center justify-center gap-2 rounded-full border-0 bg-(--m3-accent) px-5 text-[13px]/[18px] tracking-[0.1px] font-medium text-(--m3-on-accent) transition-[background] duration-140 ease-[ease] disabled:opacity-45 [&_svg]:size-4 [&_svg]:fill-current";
    case "add":
      return "inline-flex h-8 items-center gap-2 self-end rounded-full border border-(--m3-outline-strong) bg-transparent py-0 pr-[18px] pl-3.5 text-[13px]/[18px] tracking-[0.1px] font-medium text-(--m3-accent) transition-[background,border-color] duration-140 ease-[ease] hover:border-(--m3-accent) hover:bg-(--m3-state) [&_svg]:size-4 [&_svg]:fill-current";
    case "link":
      return props.active
        ? "inline-flex h-8 items-center justify-center rounded-full border border-transparent bg-(--m3-accent) px-4 text-[13px]/[18px] tracking-[0.1px] font-medium text-(--m3-on-accent) transition-[background,border-color] duration-140 ease-[ease]"
        : "inline-flex h-8 items-center justify-center rounded-full border border-(--m3-outline-strong) bg-transparent px-4 text-[13px]/[18px] tracking-[0.1px] font-medium text-(--m3-accent) transition-[background,border-color] duration-140 ease-[ease] not-disabled:hover:border-(--m3-accent) not-disabled:hover:bg-(--m3-state) disabled:opacity-[0.38]";
    default:
      return "inline-flex h-[34px] items-center justify-center gap-2 rounded-full border-0 bg-transparent px-5 text-[13px]/[18px] tracking-[0.1px] font-medium text-(--m3-accent) transition-[background] duration-140 ease-[ease] hover:bg-(--m3-state) disabled:opacity-45 [&_svg]:size-4 [&_svg]:fill-current";
  }
});
</script>

<template>
  <button :class="classes" type="button" :disabled="disabled">
    <slot />
  </button>
</template>
