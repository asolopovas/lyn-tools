export type Project = {
  name: string;
  path: string;
  kind: string;
  distro?: string;
  displayName?: string;
  detectedAt: string;
  usageCount: number;
  lastLaunchedAt: string;
};

export type ScanResult = {
  count: number;
  error?: string;
};

export type LaunchAction = "open" | "code" | "terminal";
export type ProjectAction = LaunchAction | "reveal" | "run-admin" | "run-user";

export type LaunchRequest = {
  path: string;
  action: ProjectAction;
  distro?: string | undefined;
};

export type WSLRoot = {
  distro?: string;
  path: string;
};

export type LaunchResult = {
  command: string;
  args: string[];
  error?: string;
};

export type ElevationMode = "standard" | "admin";
export type WindowMode = "launcher" | "settings";

export type ElevationStatus = {
  mode: ElevationMode;
  canSwitch: boolean;
  message?: string;
};

export type Theme = {
  name: string;
  background: string;
  panel: string;
  panelAlt: string;
  border: string;
  text: string;
  muted: string;
  accent: string;
  selected: string;
};

export type LynConfig = {
  path: string;
  cache: {
    dir: string;
  };
  startup: {
    enabled: boolean;
    startHidden: boolean;
  };
  scanner: {
    roots: string[];
    wslRoots?: WSLRoot[];
    maxDepth: number;
    concurrency: number;
    timeout: string;
    watch: boolean;
  };
  hotkey: {
    binding: string;
  };
  ui: {
    theme: string;
    backgroundOpacity: number;
    selectionColor: string;
    windowPlacement: "center";
    clearQueryOnShow: boolean;
    workspaceQueryShortcut: string;
  };
};

export type CachedLauncherState = {
  version: 1;
  cfg: LynConfig;
  projects: Project[];
  projectIcons: Record<string, string>;
};

export type WailsApp = {
  Config: () => Promise<LynConfig>;
  SaveConfig: (cfg: LynConfig) => Promise<LynConfig>;
  Projects: () => Promise<Project[]>;
  RefreshProjects: () => Promise<Project[]>;
  SearchProjects: (query: string) => Promise<Project[]>;
  Scan: () => Promise<ScanResult>;
  Icon: (path: string) => Promise<string>;
  Launch: (req: LaunchRequest) => Promise<LaunchResult>;
  SetLaunchSelection: (req: LaunchRequest) => Promise<void>;
  Debug: (stage: string, detail: string) => Promise<void>;
  ChooseFolder: () => Promise<string>;
  ChooseWSLFolder: () => Promise<WSLRoot>;
  WSLDistros: () => Promise<string[]>;
  ElevationStatus: () => Promise<ElevationStatus>;
  SwitchElevation: (mode: ElevationMode) => Promise<ElevationStatus>;
  Hide: () => Promise<void>;
  WindowMode: () => Promise<WindowMode>;
  OpenSettingsWindow: () => Promise<void>;
  CloseSettingsWindow: () => Promise<void>;
};
