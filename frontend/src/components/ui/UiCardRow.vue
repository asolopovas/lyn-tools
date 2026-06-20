<script setup lang="ts">
withDefaults(
  defineProps<{
    as?: "div" | "label";
  }>(),
  { as: "div" },
);
</script>

<template>
  <component
    :is="as"
    class="relative px-3.5 py-2.5 after:absolute after:inset-x-3.5 after:bottom-0 after:h-px after:bg-(--m3-outline) after:content-[''] last:after:hidden"
  >
    <slot />
  </component>
</template>
