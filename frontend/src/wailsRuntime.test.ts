import { afterEach, describe, expect, it, vi } from "vitest";
import { EventsEmit, EventsOn, WindowCenter, WindowSetSize } from "./wailsRuntime";

function mockRuntime(): {
  dispose: ReturnType<typeof vi.fn>;
  runtime: {
    EventsEmit: ReturnType<typeof vi.fn>;
    EventsOn: ReturnType<typeof vi.fn>;
    WindowCenter: ReturnType<typeof vi.fn>;
    WindowSetSize: ReturnType<typeof vi.fn>;
  };
} {
  const dispose = vi.fn();
  const runtime = {
    EventsEmit: vi.fn(),
    EventsOn: vi.fn(() => dispose),
    WindowCenter: vi.fn(),
    WindowSetSize: vi.fn(),
  };
  vi.stubGlobal("runtime", runtime);
  return { dispose, runtime };
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("wailsRuntime", () => {
  it("routes event and window calls through the injected Wails runtime", () => {
    const { dispose, runtime } = mockRuntime();
    const listener = vi.fn();

    EventsEmit("launcher-ready", 7);
    const receivedDispose = EventsOn("launcher-shown", listener);
    WindowSetSize(640, 306);
    WindowCenter();

    expect(runtime.EventsEmit).toHaveBeenCalledExactlyOnceWith("launcher-ready", 7);
    expect(runtime.EventsOn).toHaveBeenCalledExactlyOnceWith("launcher-shown", listener);
    expect(receivedDispose).toBe(dispose);
    expect(runtime.WindowSetSize).toHaveBeenCalledExactlyOnceWith(640, 306);
    expect(runtime.WindowCenter).toHaveBeenCalledOnce();
  });

  it("fails loudly when the Wails runtime is missing", () => {
    expect(() => WindowCenter()).toThrow("Wails runtime is unavailable");
  });
});
