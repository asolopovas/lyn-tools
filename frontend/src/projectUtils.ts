import type { Project } from "./types";

export function title(project: Project): string {
  if (project.displayName) return project.displayName;
  if (project.kind === "vscode-workspace") {
    return `${project.name} (${remoteLabel(project)}Workspace)`;
  }
  if (project.kind === "vscode-recent") {
    return `${project.name} (${remoteLabel(project)}Recent)`;
  }
  return project.name;
}

export function detail(project: Project): string {
  if (project.kind === "app" || project.kind === "system-command") {
    return "";
  }
  return tailSegments(project.path, 3);
}

function tailSegments(path: string, count: number): string {
  const segments = path.split(/[\\/]+/).filter((segment) => segment.length > 0);
  return segments.slice(-count).join("/");
}

function remoteLabel(project: Project): string {
  return isSSHRemoteProject(project) ? "SSH " : "";
}

function isSSHRemoteProject(project: Project): boolean {
  const path = project.path.toLowerCase();
  return (
    path.startsWith("vscode-remote://ssh-remote+") ||
    path.startsWith("vscode-remote://ssh-remote%2b")
  );
}
