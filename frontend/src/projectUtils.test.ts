import { describe, expect, it } from "vitest";
import { detail, title } from "./projectUtils";
import type { Project } from "./types";

const baseProject: Project = {
  name: "example",
  path: "",
  kind: "vscode-workspace",
  detectedAt: "",
  usageCount: 0,
  lastLaunchedAt: "",
};

function project(path: string, kind = "vscode-workspace", displayName?: string): Project {
  const item = { ...baseProject, path, kind };
  return displayName === undefined ? item : { ...item, displayName };
}

describe("project title", () => {
  it.each([
    ["local workspace", project("/home/me/example.code-workspace"), "example (Workspace)"],
    [
      "SSH workspace",
      project("vscode-remote://ssh-remote+host/srv/www/example.code-workspace"),
      "example (SSH Workspace)",
    ],
    [
      "encoded SSH recent",
      project("vscode-remote://ssh-remote%2Bhost/srv/www/example", "vscode-recent"),
      "example (SSH Recent)",
    ],
    [
      "VS Code display name",
      project(
        "vscode-remote://ssh-remote+host/srv/www/example.code-workspace",
        "vscode-workspace",
        "example (Workspace) [SSH: host]",
      ),
      "example (Workspace) [SSH: host]",
    ],
  ] as const)("formats %s", (_label, item, expected) => {
    expect(title(item)).toBe(expected);
  });
});

describe("project detail", () => {
  it.each([
    [
      "trims a deep path to its last 3 segments",
      project("/home/me/src/acme/website", "node"),
      "src/acme/website",
    ],
    ["keeps short paths intact", project("/srv/example", "laravel"), "srv/example"],
    ["handles Windows separators", project("C:\\Users\\me\\src\\acme\\app", "go"), "src/acme/app"],
    ["omits the line for apps", project("/usr/bin/firefox", "app"), ""],
    ["omits the line for system commands", project("lyn:system:logout", "system-command"), ""],
  ] as const)("%s", (_label, item, expected) => {
    expect(detail(item)).toBe(expected);
  });
});
