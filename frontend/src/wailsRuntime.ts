type WailsRuntime = {
  EventsEmit?: (eventName: string, ...optionalData: unknown[]) => void;
  EventsOn?: (eventName: string, callback: (...data: unknown[]) => void) => () => void;
  WindowCenter?: () => void;
  WindowSetSize?: (width: number, height: number) => void;
};

type WailsRuntimeGlobal = typeof globalThis & {
  runtime?: WailsRuntime;
};

function wailsRuntime(): WailsRuntime {
  const runtime = (globalThis as WailsRuntimeGlobal).runtime;
  if (!runtime) {
    throw new Error("Wails runtime is unavailable");
  }
  return runtime;
}

export function EventsEmit(eventName: string, ...optionalData: unknown[]): void {
  const emit = wailsRuntime().EventsEmit;
  if (!emit) {
    throw new Error("Wails runtime EventsEmit is unavailable");
  }
  emit(eventName, ...optionalData);
}

export function EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void {
  const on = wailsRuntime().EventsOn;
  if (!on) {
    throw new Error("Wails runtime EventsOn is unavailable");
  }
  return on(eventName, callback);
}

export function WindowCenter(): void {
  const center = wailsRuntime().WindowCenter;
  if (!center) {
    throw new Error("Wails runtime WindowCenter is unavailable");
  }
  center();
}

export function WindowSetSize(width: number, height: number): void {
  const setSize = wailsRuntime().WindowSetSize;
  if (!setSize) {
    throw new Error("Wails runtime WindowSetSize is unavailable");
  }
  setSize(width, height);
}
