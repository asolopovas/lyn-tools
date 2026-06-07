import { computed, ref, watch } from "vue";
import { backend } from "./backend";
import { errorMessage } from "./errors";
import { readLauncherCache, writeLauncherCache } from "./launcherCache";
import type { Project, WailsApp } from "./types";

export function useLauncherState(api: WailsApp = backend) {
  const cached = readLauncherCache();
  const query = ref("");
  const cfg = ref(cached?.cfg ?? null);
  const projects = ref<Project[]>(cached?.projects ?? []);
  const matches = ref<Project[]>([]);
  const projectIcons = ref<Record<string, string>>(cached?.projectIcons ?? {});
  const selectedIndex = ref(0);
  const loading = ref(true);
  const scanning = ref(false);
  const status = ref("Ready");
  const selectedProject = computed(() => matches.value[selectedIndex.value] ?? null);
  const statusLine = computed(() => {
    if (status.value !== "Ready") {
      return status.value;
    }
    return cfg.value ? `${projects.value.length} projects` : "Loading";
  });
  let initialLoad: Promise<void> | null = null;
  let searchSequence = 0;

  watch(query, () => {
    void updateMatches(true);
  });

  function cacheState(): void {
    writeLauncherCache(cfg.value, projects.value, projectIcons.value);
  }

  async function loadConfigOnly(): Promise<void> {
    cfg.value = await api.Config();
    cacheState();
    loading.value = false;
  }

  async function refresh(live = false): Promise<void> {
    cfg.value = await api.Config();
    projects.value = live ? await api.RefreshProjects() : await api.Projects();
    await updateMatches();
    cacheState();
  }

  async function updateMatches(resetSelection = false): Promise<void> {
    const sequence = ++searchSequence;
    const currentPath = selectedProject.value?.path;
    const currentIndex = selectedIndex.value;
    try {
      const found = await api.SearchProjects(query.value);
      if (sequence !== searchSequence) {
        return;
      }
      selectedIndex.value = nextSelectionIndex(found, currentPath, currentIndex, resetSelection);
      matches.value = found;
    } catch (error) {
      status.value = errorMessage(error, "Search failed");
    }
  }

  function ensureInitialProjects(): Promise<void> {
    initialLoad ??= loadInitialProjects();
    return initialLoad;
  }

  async function loadInitialProjects(): Promise<void> {
    try {
      await refresh();
      await loadVisibleIcons();
    } catch (error) {
      status.value = errorMessage(error, "Failed to load projects");
    } finally {
      loading.value = false;
    }
  }

  async function refreshFromWatcher(): Promise<void> {
    await refresh();
    status.value = "Index updated";
  }

  async function refreshForShow(): Promise<void> {
    try {
      await refresh(true);
      await loadVisibleIcons();
    } catch (error) {
      status.value = errorMessage(error, "Failed to refresh projects");
    }
  }

  async function loadVisibleIcons(): Promise<void> {
    const missing = matches.value.filter(
      (project) => project.kind === "app" && projectIcons.value[project.path] === undefined,
    );
    if (!missing.length) {
      return;
    }
    const loaded = await Promise.all(
      missing.map(async (project) => {
        try {
          return [project.path, await api.Icon(project.path)] as const;
        } catch {
          return [project.path, ""] as const;
        }
      }),
    );
    projectIcons.value = {
      ...projectIcons.value,
      ...Object.fromEntries(loaded),
    };
    cacheState();
  }

  async function scan(): Promise<void> {
    scanning.value = true;
    status.value = "Scanning";
    try {
      const result = await api.Scan();
      status.value = result.error ? result.error : `Indexed ${result.count} projects`;
      await refresh();
    } catch (error) {
      status.value = errorMessage(error, "Scan failed");
    } finally {
      scanning.value = false;
    }
  }

  function moveSelection(delta: number): void {
    if (!matches.value.length) {
      selectedIndex.value = 0;
      return;
    }
    const nextIndex = selectedIndex.value + delta;
    selectedIndex.value = Math.min(Math.max(nextIndex, 0), matches.value.length - 1);
  }

  return {
    query,
    cfg,
    projects,
    matches,
    projectIcons,
    selectedIndex,
    loading,
    scanning,
    status,
    selectedProject,
    statusLine,
    cacheState,
    ensureInitialProjects,
    loadConfigOnly,
    loadVisibleIcons,
    refreshForShow,
    refreshFromWatcher,
    scan,
    updateMatches,
    moveSelection,
  };
}

function nextSelectionIndex(
  found: Project[],
  currentPath: string | undefined,
  currentIndex: number,
  resetSelection: boolean,
): number {
  if (!found.length || resetSelection) {
    return 0;
  }
  const pathIndex = currentPath ? found.findIndex((project) => project.path === currentPath) : -1;
  if (pathIndex >= 0) {
    return pathIndex;
  }
  return Math.min(Math.max(currentIndex, 0), found.length - 1);
}
