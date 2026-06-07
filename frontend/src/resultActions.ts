import { isPackagedWindowsApp } from "./launchActions";
import type { Project, ProjectAction } from "./types";

export type ResultAction = {
  action: ProjectAction;
  glyph: string;
  title: string;
};

const projectActions: ResultAction[] = [
  { action: "code", glyph: "\uE8A7", title: "Open in VS Code (Enter)" },
  { action: "reveal", glyph: "\uE838", title: "Open containing folder (Ctrl+Shift+E)" },
  { action: "terminal", glyph: "\uE756", title: "Open terminal here (Ctrl+Shift+C)" },
];

const appActions: ResultAction[] = [
  { action: "run-admin", glyph: "\uE7EF", title: "Run as administrator (Ctrl+Shift+Enter)" },
  { action: "run-user", glyph: "\uE7EE", title: "Run as different user (Ctrl+Shift+U)" },
  { action: "reveal", glyph: "\uE838", title: "Open containing folder (Ctrl+Shift+E)" },
  {
    action: "terminal",
    glyph: "\uE756",
    title: "Open terminal in containing folder (Ctrl+Shift+C)",
  },
];

export function isApp(project: Project): boolean {
  return project.kind === "app";
}

export function isSystemCommand(project: Project): boolean {
  return project.kind === "system-command";
}

export function systemCommandGlyph(project: Project): string {
  switch (project.path) {
    case "lyn:system:restart":
      return "\uE777";
    case "lyn:system:shutdown":
      return "\uE7E8";
    case "lyn:system:logout":
      return "\uF3B1";
    default:
      return "\uE768";
  }
}

export function actionButtons(project: Project): ResultAction[] {
  if (isSystemCommand(project)) {
    return [
      { action: "open", glyph: systemCommandGlyph(project), title: "Run system command (Enter)" },
    ];
  }
  if (isPackagedWindowsApp(project)) {
    return [{ action: "open", glyph: "\uE8A7", title: "Open app (Enter)" }];
  }
  return isApp(project) ? appActions : projectActions;
}
