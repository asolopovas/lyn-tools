import { afterEach, describe, expect, it, vi } from "vitest";
import { backend } from "./backend";
import type { ProjectAction } from "./types";

type MockBinding = {
  ChooseFolder: ReturnType<typeof vi.fn>;
  Config: ReturnType<typeof vi.fn>;
  ElevationStatus: ReturnType<typeof vi.fn>;
  Debug: ReturnType<typeof vi.fn>;
  Hide: ReturnType<typeof vi.fn>;
  Icon: ReturnType<typeof vi.fn>;
  Launch: ReturnType<typeof vi.fn>;
  Projects: ReturnType<typeof vi.fn>;
  RefreshProjects: ReturnType<typeof vi.fn>;
  SaveConfig: ReturnType<typeof vi.fn>;
  Scan: ReturnType<typeof vi.fn>;
  SearchProjects: ReturnType<typeof vi.fn>;
  SetLaunchSelection: ReturnType<typeof vi.fn>;
  SwitchElevation: ReturnType<typeof vi.fn>;
  WindowMode: ReturnType<typeof vi.fn>;
  OpenSettingsWindow: ReturnType<typeof vi.fn>;
  CloseSettingsWindow: ReturnType<typeof vi.fn>;
};

function mockAppBinding(namespace: "main" | "lyn" = "main"): MockBinding {
  const app = {
    ChooseFolder: vi.fn(),
    Config: vi.fn(),
    Debug: vi.fn(),
    ElevationStatus: vi.fn(),
    Hide: vi.fn(),
    Icon: vi.fn(),
    Launch: vi.fn(),
    Projects: vi.fn(),
    RefreshProjects: vi.fn(),
    SaveConfig: vi.fn(),
    Scan: vi.fn(),
    SearchProjects: vi.fn(),
    SetLaunchSelection: vi.fn(),
    SwitchElevation: vi.fn(),
    WindowMode: vi.fn(),
    OpenSettingsWindow: vi.fn(),
    CloseSettingsWindow: vi.fn(),
  };
  vi.stubGlobal("go", {
    [namespace]: { App: app },
  });
  return app;
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("backend", () => {
  it("routes launcher actions through the generated main Wails binding", async () => {
    const app = mockAppBinding("main");
    app.Launch.mockResolvedValue({ command: "code", args: [] });
    await expect(backend.Launch({ path: "C:/src/app", action: "code" })).resolves.toMatchObject({
      command: "code",
    });
    expect(app.Launch).toHaveBeenCalledExactlyOnceWith({ path: "C:/src/app", action: "code" });
  });

  it("routes launcher actions through the lyn Wails binding", async () => {
    const app = mockAppBinding("lyn");
    app.Launch.mockResolvedValue({ command: "explorer.exe", args: [] });
    await backend.Launch({ path: "shell:AppsFolder\\WhatsApp!App", action: "open" });
    expect(app.Launch).toHaveBeenCalledExactlyOnceWith({
      path: "shell:AppsFolder\\WhatsApp!App",
      action: "open" as ProjectAction,
    });
  });

  it("routes search through the Wails binding", async () => {
    const app = mockAppBinding();
    app.SearchProjects.mockResolvedValue([{ name: "Calculator", path: "/calculator" }]);
    await expect(backend.SearchProjects("calclator")).resolves.toHaveLength(1);
    expect(app.SearchProjects).toHaveBeenCalledExactlyOnceWith("calclator");
  });

  it("routes selected launch fallback state through the Wails binding", async () => {
    const app = mockAppBinding();
    app.SetLaunchSelection.mockResolvedValue(undefined);
    await backend.SetLaunchSelection({ path: "C:/src/app", action: "code" });
    expect(app.SetLaunchSelection).toHaveBeenCalledExactlyOnceWith({
      path: "C:/src/app",
      action: "code",
    });
  });

  it("routes frontend diagnostics through the Wails binding", async () => {
    const app = mockAppBinding();
    app.Debug.mockResolvedValue(undefined);
    await backend.Debug("hotkey", "Enter");
    expect(app.Debug).toHaveBeenCalledExactlyOnceWith("hotkey", "Enter");
  });

  it("routes elevation controls through the Wails binding", async () => {
    const app = mockAppBinding();
    app.ElevationStatus.mockResolvedValue({ mode: "standard", canSwitch: true });
    app.SwitchElevation.mockResolvedValue({ mode: "admin", canSwitch: true });
    await expect(backend.ElevationStatus()).resolves.toMatchObject({ mode: "standard" });
    await expect(backend.SwitchElevation("admin")).resolves.toMatchObject({ mode: "admin" });
    expect(app.ElevationStatus).toHaveBeenCalledOnce();
    expect(app.SwitchElevation).toHaveBeenCalledExactlyOnceWith("admin");
  });

  it("routes settings window controls through the Wails binding", async () => {
    const app = mockAppBinding();
    app.WindowMode.mockResolvedValue("settings");
    app.OpenSettingsWindow.mockResolvedValue(undefined);
    app.CloseSettingsWindow.mockResolvedValue(undefined);
    await expect(backend.WindowMode()).resolves.toBe("settings");
    await backend.OpenSettingsWindow();
    await backend.CloseSettingsWindow();
    expect(app.WindowMode).toHaveBeenCalledOnce();
    expect(app.OpenSettingsWindow).toHaveBeenCalledOnce();
    expect(app.CloseSettingsWindow).toHaveBeenCalledOnce();
  });
});
