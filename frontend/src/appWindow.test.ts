import { ref } from "vue";
import { describe, expect, test, vi } from "vitest";
import { useAppWindow } from "./appWindow";
import type { WailsApp, WindowMode } from "./types";

function setup(mode: WindowMode = "launcher") {
  const listeners = new Map<string, EventListener>();
  const timers = new Map<number, () => void>();
  let nextTimer = 1;
  const host = {
    addEventListener: vi.fn((name: string, callback: EventListener) =>
      listeners.set(name, callback),
    ),
    removeEventListener: vi.fn((name: string) => listeners.delete(name)),
    setInterval: vi.fn((callback: () => void) => {
      const id = nextTimer++;
      timers.set(id, callback);
      return id;
    }),
    clearInterval: vi.fn((id: number) => timers.delete(id)),
    setTimeout: vi.fn((callback: () => void) => {
      const id = nextTimer++;
      timers.set(id, callback);
      return id;
    }),
    clearTimeout: vi.fn((id: number) => timers.delete(id)),
  };
  const api = {
    WindowMode: vi.fn(async () => mode),
    Platform: vi.fn(async () => "windows"),
    Hide: vi.fn(async () => undefined),
    Debug: vi.fn(async () => undefined),
    OpenSettingsWindow: vi.fn(async () => undefined),
    CloseSettingsWindow: vi.fn(async () => undefined),
    SetLaunchSelection: vi.fn(async () => undefined),
  } as unknown as WailsApp;
  const eventCallbacks = new Map<string, (...data: unknown[]) => void>();
  const disposers: Array<ReturnType<typeof vi.fn>> = [];
  const runtime = {
    emit: vi.fn(),
    on: vi.fn((name: string, callback: (...data: unknown[]) => void) => {
      eventCallbacks.set(name, callback);
      const dispose = vi.fn();
      disposers.push(dispose);
      return dispose;
    }),
    center: vi.fn(),
    setSize: vi.fn(),
  };
  const callbacks = {
    ensureInitialProjects: vi.fn(async () => undefined),
    loadConfigOnly: vi.fn(async () => undefined),
    loadElevationStatus: vi.fn(async () => undefined),
    loadWSLDistros: vi.fn(async () => undefined),
    focusQuery: vi.fn(),
    refreshForShow: vi.fn(async () => undefined),
    refreshFromWatcher: vi.fn(async () => undefined),
    updateNativeLaunchSelection: vi.fn(),
  };
  const appWindow = useAppWindow({
    api,
    cfg: ref(null),
    projects: ref([]),
    query: ref("query"),
    selectedIndex: ref(3),
    status: ref("Ready"),
    blurHideSuppressed: ref(false),
    ...callbacks,
    host: host as unknown as Window,
    hasFocus: () => false,
    runtime,
  });
  return { appWindow, api, host, listeners, timers, runtime, eventCallbacks, disposers, callbacks };
}

describe("app window", () => {
  test("registers launcher events and releases listeners, polling, and subscriptions", async () => {
    const { appWindow, api, host, runtime, disposers } = setup();
    await appWindow.mount();

    expect(host.addEventListener).toHaveBeenCalledWith("focus", expect.any(Function));
    expect(host.addEventListener).toHaveBeenCalledWith("blur", appWindow.onWindowBlur);
    expect(host.setInterval).toHaveBeenCalledWith(expect.any(Function), 100);
    expect(runtime.on).toHaveBeenCalledTimes(2);

    appWindow.unmount();

    expect(host.clearInterval).toHaveBeenCalledOnce();
    expect(api.SetLaunchSelection).toHaveBeenCalledWith({ path: "", action: "code" });
    expect(disposers.every((dispose) => dispose.mock.calls.length === 1)).toBe(true);
  });

  test("sizes the launcher and acknowledges readiness after preparing data", async () => {
    const { appWindow, runtime, eventCallbacks, callbacks } = setup();
    await appWindow.mount();
    expect(runtime.setSize).toHaveBeenCalledWith(640, 306);

    eventCallbacks.get("launcher-shown")?.(42);
    await Promise.resolve();
    await Promise.resolve();

    expect(callbacks.ensureInitialProjects).toHaveBeenCalled();
    expect(runtime.emit).toHaveBeenCalledWith("launcher-ready", 42);
    expect(callbacks.refreshForShow).toHaveBeenCalled();
    appWindow.unmount();
  });

  test("hides after an unsuppressed blur and clears its timer", async () => {
    const { appWindow, api, listeners, timers } = setup();
    await appWindow.mount();
    listeners.get("blur")?.(new Event("blur"));
    const callback = [...timers.values()].at(-1);
    callback?.();
    await Promise.resolve();

    expect(api.Debug).toHaveBeenCalledWith("window.blur", "hide");
    expect(api.Hide).toHaveBeenCalled();
    appWindow.unmount();
  });

  test("initializes a settings window without launcher polling", async () => {
    const { appWindow, host, runtime, callbacks } = setup("settings");
    await appWindow.mount();

    expect(callbacks.loadConfigOnly).toHaveBeenCalledOnce();
    expect(callbacks.loadElevationStatus).toHaveBeenCalled();
    expect(runtime.setSize).toHaveBeenCalledWith(720, 650);
    expect(host.setInterval).not.toHaveBeenCalled();
    expect(appWindow.appReady.value).toBe(true);
    appWindow.unmount();
  });
});
