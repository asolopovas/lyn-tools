import { describe, expect, it } from "vitest";
import { title } from "./projectUtils";
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
