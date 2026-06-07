import type { Project } from "./types";

const kindLabels: Record<string, string> = {
  wordpress: "WordPress",
  laravel: "Laravel",
  go: "Go Module",
  node: "Node Package",
  rust: "Rust Crate",
  "vscode-workspace": "Workspace",
  "vscode-recent": "VS Code Recent",
  app: "App",
  "system-command": "System Command",
};

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
  return `${kindLabel(project.kind)}: ${project.path}`;
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

function kindLabel(kind: string): string {
  return kindLabels[kind] ?? "Project Folder";
}
