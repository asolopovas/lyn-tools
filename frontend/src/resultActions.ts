import type { IconName } from "./icons";
import { isPackagedWindowsApp } from "./launchActions";
import type { Project, ProjectAction } from "./types";

export type ResultAction = {
  action: ProjectAction;
  icon: IconName;
  title: string;
};

const projectActions: ResultAction[] = [
  { action: "code", icon: "code", title: "Open in VS Code (Enter)" },
  { action: "reveal", icon: "folderOpen", title: "Open containing folder (Ctrl+Shift+E)" },
  { action: "terminal", icon: "terminal", title: "Open terminal here (Ctrl+Shift+C)" },
];

const appActions: ResultAction[] = [
  { action: "run-admin", icon: "shieldAccount", title: "Run as administrator (Ctrl+Shift+Enter)" },
  { action: "run-user", icon: "account", title: "Run as different user (Ctrl+Shift+U)" },
  { action: "reveal", icon: "folderOpen", title: "Open containing folder (Ctrl+Shift+E)" },
  {
    action: "terminal",
    icon: "terminal",
    title: "Open terminal in containing folder (Ctrl+Shift+C)",
  },
];

export function isApp(project: Project): boolean {
  return project.kind === "app";
}

export function isSystemCommand(project: Project): boolean {
  return project.kind === "system-command";
}

export function systemCommandIcon(project: Project): IconName {
  switch (project.path) {
    case "lyn:system:restart":
      return "restart";
    case "lyn:system:shutdown":
      return "power";
    case "lyn:system:logout":
      return "logout";
    default:
      return "play";
  }
}

export function actionButtons(project: Project): ResultAction[] {
  if (isSystemCommand(project)) {
    return [
      { action: "open", icon: systemCommandIcon(project), title: "Run system command (Enter)" },
    ];
  }
  if (isPackagedWindowsApp(project)) {
    return [{ action: "open", icon: "openInNew", title: "Open app (Enter)" }];
  }
  return isApp(project) ? appActions : projectActions;
}
