import { ref } from "vue";
import { backend } from "./backend";
import { errorMessage } from "./errors";
import { actionForProject, defaultAction } from "./launchActions";
import type { Project, ProjectAction, WailsApp } from "./types";
import type { ComputedRef, Ref } from "vue";

export function useLauncherLaunch(options: {
  api?: WailsApp;
  settingsOpen: Ref<boolean>;
  status: Ref<string>;
  query: Ref<string>;
  matches: Ref<Project[]>;
  selectedIndex: Ref<number>;
  selectedProject: ComputedRef<Project | null>;
  hideLauncher: () => Promise<void>;
}) {
  const api = options.api ?? backend;
  const launchInFlight = ref(false);
  let lastLaunchSignature = "";
  let lastLaunchAt = 0;
  let nativeSelectionSignature = "";

  async function launch(project: Project, action: ProjectAction, hide = true): Promise<void> {
    const signature = `${action}\u0000${project.path}`;
    const now = performance.now();
    if (launchInFlight.value || (signature === lastLaunchSignature && now - lastLaunchAt < 700)) {
      void api.Debug("launch.skipped", launchInFlight.value ? "in-flight" : "duplicate");
      return;
    }
    lastLaunchSignature = signature;
    lastLaunchAt = now;
    launchInFlight.value = true;
    void api.Debug("launch.request", `${action} ${project.kind} ${project.path}`);
    try {
      const result = await api.Launch({ path: project.path, action, distro: project.distro });
      if (result.error) {
        options.status.value = result.error;
        return;
      }
      options.status.value = `Started ${result.command}`;
      if (hide) {
        options.query.value = "";
        options.selectedIndex.value = 0;
        await options.hideLauncher();
      }
    } catch (error) {
      options.status.value = errorMessage(error, "Launch failed");
      void api.Debug("launch.error", options.status.value);
    } finally {
      launchInFlight.value = false;
    }
  }

  function updateNativeLaunchSelection(): void {
    const project = options.selectedProject.value;
    const action =
      project && !options.settingsOpen.value ? actionForProject(project, "code") : "code";
    const path = project && !options.settingsOpen.value ? project.path : "";
    const distro = project && !options.settingsOpen.value ? project.distro : undefined;
    const signature = `${action}\u0000${path}`;
    if (signature === nativeSelectionSignature) {
      return;
    }
    nativeSelectionSignature = signature;
    void api.SetLaunchSelection({ path, action, distro });
  }

  function launchSelected(action: ProjectAction): void {
    const project = options.selectedProject.value;
    if (!project) {
      void api.Debug(
        "launch.selected.missing",
        `action=${action} query=${options.query.value} matches=${options.matches.value.length}`,
      );
      return;
    }
    const mappedAction = actionForProject(project, action);
    void api.Debug("launch.selected", `${mappedAction} ${project.kind} ${project.path}`);
    void launch(project, mappedAction, mappedAction !== "reveal");
  }

  function launchDefault(project: Project): void {
    void launch(project, defaultAction(project));
  }

  function launchAtIndex(index: number): void {
    const project = options.matches.value[index];
    if (!project) {
      void api.Debug(
        "launch.index.missing",
        `index=${index} matches=${options.matches.value.length}`,
      );
      return;
    }
    options.selectedIndex.value = index;
    launchDefault(project);
  }

  function launchProjectAction(project: Project, action: ProjectAction): void {
    void launch(project, action, action !== "reveal");
  }

  return {
    launchAtIndex,
    launchDefault,
    launchProjectAction,
    launchSelected,
    updateNativeLaunchSelection,
  };
}
