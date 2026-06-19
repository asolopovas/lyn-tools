import { describe, expect, it } from "vitest";
import { actionButtons, isSystemCommand, systemCommandIcon } from "./resultActions";
import type { Project } from "./types";

function project(kind: string, path = `C:/${kind}`): Project {
  return {
    name: kind,
    path,
    kind,
    detectedAt: "",
    usageCount: 0,
    lastLaunchedAt: "",
  };
}

describe("result actions", () => {
  it("uses project actions for folders", () => {
    expect(actionButtons(project("go")).map((button) => button.action)).toEqual([
      "code",
      "reveal",
      "terminal",
    ]);
  });

  it("uses app actions for regular apps and open-only for packaged apps", () => {
    expect(actionButtons(project("app")).map((button) => button.action)).toEqual([
      "run-admin",
      "run-user",
      "reveal",
      "terminal",
    ]);
    expect(actionButtons(project("app", "shell:AppsFolder\\WhatsApp!App"))).toEqual([
      { action: "open", icon: "openInNew", title: "Open app (Enter)" },
    ]);
  });

  it("uses system command icons", () => {
    const restart = project("system-command", "lyn:system:restart");
    expect(isSystemCommand(restart)).toBe(true);
    expect(systemCommandIcon(restart)).toBe("restart");
    expect(systemCommandIcon(project("system-command", "lyn:system:shutdown"))).toBe("power");
    expect(systemCommandIcon(project("system-command", "lyn:system:logout"))).toBe("logout");
    expect(actionButtons(restart)).toEqual([
      { action: "open", icon: "restart", title: "Run system command (Enter)" },
    ]);
  });
});
