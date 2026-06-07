import type { Project, ProjectAction } from "./types";

export function defaultAction(project: Project): ProjectAction {
  return project.kind === "app" || project.kind === "system-command" ? "open" : "code";
}

export function actionForProject(project: Project, action: ProjectAction): ProjectAction {
  if (project.kind === "system-command" || isPackagedWindowsApp(project)) {
    return "open";
  }
  if (project.kind === "app" && action === "code") {
    return "open";
  }
  if (project.kind !== "app" && (action === "run-admin" || action === "run-user")) {
    return "code";
  }
  return action;
}

export function isPackagedWindowsApp(project: Project): boolean {
  return project.kind === "app" && project.path.startsWith("shell:AppsFolder\\");
}
