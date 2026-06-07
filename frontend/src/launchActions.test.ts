import { describe, expect, it } from "vitest";
import { actionForProject, defaultAction, isPackagedWindowsApp } from "./launchActions";
import type { Project } from "./types";

function project(kind: string): Project {
  return {
    name: kind,
    path: `C:/${kind}`,
    kind,
    detectedAt: "",
    usageCount: 0,
    lastLaunchedAt: "",
  };
}

describe("launch actions", () => {
  it("opens apps and system commands by default and code projects by default", () => {
    expect(defaultAction(project("app"))).toBe("open");
    expect(defaultAction(project("system-command"))).toBe("open");
    expect(defaultAction(project("go"))).toBe("code");
  });

  it("maps unsupported actions to valid project actions", () => {
    expect(actionForProject(project("app"), "code")).toBe("open");
    expect(actionForProject(project("system-command"), "reveal")).toBe("open");
    expect(
      actionForProject({ ...project("app"), path: "shell:AppsFolder\\WhatsApp!App" }, "reveal"),
    ).toBe("open");
    expect(actionForProject(project("go"), "run-admin")).toBe("code");
    expect(actionForProject(project("go"), "terminal")).toBe("terminal");
  });

  it("detects packaged Windows apps", () => {
    expect(
      isPackagedWindowsApp({ ...project("app"), path: "shell:AppsFolder\\WhatsApp!App" }),
    ).toBe(true);
    expect(isPackagedWindowsApp(project("app"))).toBe(false);
  });
});
