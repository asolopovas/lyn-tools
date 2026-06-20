<script setup lang="ts">
defineProps<{
  title: string;
}>();
</script>

<template>
  <section class="flex flex-col gap-1.5">
    <div class="flex items-center justify-between gap-2.5">
      <h2 class="m-0 ml-1 text-[11px]/4 font-bold tracking-[0.55px] text-(--m3-accent) uppercase">
        {{ title }}
      </h2>
      <slot name="actions" />
    </div>
    <slot />
  </section>
</template>
