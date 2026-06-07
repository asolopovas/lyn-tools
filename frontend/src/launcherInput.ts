import { nextTick } from "vue";

export function useLauncherInput() {
  function queryInput(): HTMLInputElement | null {
    return document.querySelector<HTMLInputElement>("[data-lyn-query]");
  }

  function focusQuery(): void {
    void nextTick(() => {
      for (let i = 0; i < 12; i += 1) {
        window.setTimeout(() => {
          window.focus();
          queryInput()?.focus({ preventScroll: true });
        }, i * 35);
      }
    });
  }

  return { queryInput, focusQuery };
}

export function scrollSelectedResultIntoView(): void {
  void nextTick(() => {
    const list = document.querySelector<HTMLUListElement>("[data-lyn-results]");
    const selected = list?.querySelector<HTMLElement>("li.selected");
    if (!list || !selected) {
      return;
    }
    const listRect = list.getBoundingClientRect();
    const selectedRect = selected.getBoundingClientRect();
    let top = list.scrollTop;
    if (selectedRect.top < listRect.top) {
      top -= listRect.top - selectedRect.top;
    } else if (selectedRect.bottom > listRect.bottom) {
      top += selectedRect.bottom - listRect.bottom;
    }
    if (top !== list.scrollTop) {
      list.scrollTo({ top, behavior: "smooth" });
    }
  });
}
