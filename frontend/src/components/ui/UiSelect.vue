<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from "vue";
import { icons } from "../../icons";

const ITEM_H = 34;

const props = withDefaults(
  defineProps<{
    options: { value: string; label: string }[];
    maxItems?: number;
    placeholder?: string;
  }>(),
  { maxItems: 7, placeholder: "Select…" },
);

const model = defineModel<string>({ required: true });

const open = ref(false);
const focused = ref(false);
const triggerEl = ref<HTMLButtonElement | null>(null);
const menuEl = ref<HTMLUListElement | null>(null);
const rect = ref<{ left: number; top: number; width: number } | null>(null);

const selected = computed(() => props.options.find((o) => o.value === model.value) ?? null);
const menuMaxHeight = computed(() => `${props.maxItems * ITEM_H}px`);

function measure(): void {
  const el = triggerEl.value;
  if (!el) return;
  const r = el.getBoundingClientRect();
  rect.value = { left: r.left, top: r.bottom + 6, width: r.width };
}

function close(): void {
  open.value = false;
}

function toggle(): void {
  open.value = !open.value;
}

function pick(value: string): void {
  model.value = value;
  close();
}

function onDocPointerDown(event: MouseEvent): void {
  const target = event.target as Node;
  if (triggerEl.value?.contains(target)) return;
  if (menuEl.value?.contains(target)) return;
  close();
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") close();
}

function onReflow(): void {
  measure();
}

function bindGlobals(): void {
  document.addEventListener("mousedown", onDocPointerDown);
  document.addEventListener("keydown", onKeydown);
  window.addEventListener("resize", onReflow);
  window.addEventListener("scroll", onReflow, true);
}

function unbindGlobals(): void {
  document.removeEventListener("mousedown", onDocPointerDown);
  document.removeEventListener("keydown", onKeydown);
  window.removeEventListener("resize", onReflow);
  window.removeEventListener("scroll", onReflow, true);
}

watch(open, (isOpen) => {
  if (!isOpen) {
    unbindGlobals();
    return;
  }
  measure();
  bindGlobals();
  void nextTick(() => {
    menuEl.value?.querySelector("[data-selected='true']")?.scrollIntoView({ block: "nearest" });
  });
});

onUnmounted(unbindGlobals);
</script>

<template>
  <div class="relative">
    <button
      ref="triggerEl"
      type="button"
      class="flex h-9 w-full items-center justify-between gap-2 rounded-lg border bg-(--m3-surface-container-high) py-0 pr-2.5 pl-3 text-[13px]/[18px] tracking-[0.1px] text-(--m3-on-surface) outline-none transition-[border-color,box-shadow] duration-140 ease-[ease]"
      :class="
        open || focused
          ? 'border-(--m3-accent) shadow-[0_0_0_1px_var(--m3-accent)]'
          : 'border-(--m3-outline-strong)'
      "
      aria-haspopup="listbox"
      :aria-expanded="open"
      @click="toggle"
      @focus="focused = true"
      @blur="focused = false"
    >
      <span
        class="overflow-hidden text-ellipsis whitespace-nowrap"
        :class="selected ? 'text-(--m3-on-surface)' : 'text-(--m3-on-surface-variant)'"
        >{{ selected ? selected.label : placeholder }}</span
      >
      <svg
        viewBox="0 0 24 24"
        aria-hidden="true"
        class="size-[18px] flex-none fill-(--m3-on-surface-variant) transition-transform duration-140 ease-[ease]"
        :class="{ 'rotate-180': open }"
      >
        <path d="M7.4 9.6 12 14.2l4.6-4.6 1.4 1.4-6 6-6-6z" />
      </svg>
    </button>

    <Teleport to="body">
      <ul
        v-if="open && rect"
        ref="menuEl"
        role="listbox"
        class="fixed z-[1000] m-0 list-none overflow-y-auto rounded-lg border border-(--m3-outline-strong) bg-(--m3-surface-container-high) p-0 shadow-[0_12px_32px_rgba(0,0,0,.45)] [scrollbar-color:var(--m3-outline-strong)_transparent] scrollbar-thin"
        :style="{
          left: `${rect.left}px`,
          top: `${rect.top}px`,
          width: `${rect.width}px`,
          maxHeight: menuMaxHeight,
        }"
      >
        <li
          v-for="option in options"
          :key="option.value"
          role="option"
          :aria-selected="option.value === model"
          :data-selected="option.value === model"
        >
          <button
            type="button"
            class="flex h-[34px] w-full items-center justify-between gap-2 border-0 px-3 text-left text-[13px]/[18px] tracking-[0.1px]"
            :class="
              option.value === model
                ? 'bg-(--m3-accent-container) text-(--m3-accent)'
                : 'bg-transparent text-(--m3-on-surface) hover:bg-(--m3-state)'
            "
            @click="pick(option.value)"
          >
            <span class="overflow-hidden text-ellipsis whitespace-nowrap">{{ option.label }}</span>
            <svg
              v-if="option.value === model"
              viewBox="0 0 24 24"
              aria-hidden="true"
              class="size-4 flex-none fill-(--m3-accent)"
            >
              <path :d="icons.check" />
            </svg>
          </button>
        </li>
      </ul>
    </Teleport>
  </div>
</template>
