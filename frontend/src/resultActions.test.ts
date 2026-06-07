import { describe, expect, it } from "vitest";
import { actionButtons, isSystemCommand, systemCommandGlyph } from "./resultActions";
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
      { action: "open", glyph: "\uE8A7", title: "Open app (Enter)" },
    ]);
  });

  it("uses system command glyphs", () => {
    const restart = project("system-command", "lyn:system:restart");
    expect(isSystemCommand(restart)).toBe(true);
    expect(systemCommandGlyph(restart)).toBe("\uE777");
    expect(actionButtons(restart)).toEqual([
      { action: "open", glyph: "\uE777", title: "Run system command (Enter)" },
    ]);
  });
});
