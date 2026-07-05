import type { WailsApp } from "./types";

type WailsGlobal = typeof globalThis & {
  go?: {
    main?: { App?: WailsApp };
    lyn?: { App?: WailsApp };
  };
};

function appBinding(): WailsApp {
  const go = (globalThis as WailsGlobal).go;
  const app = go?.main?.App ?? go?.lyn?.App;
  if (!app) {
    throw new Error("Wails App binding is unavailable");
  }
  return app;
}

export const backend: WailsApp = {
  Platform: () => appBinding().Platform(),
  Config: () => appBinding().Config(),
  SaveConfig: (cfg) => appBinding().SaveConfig(cfg),
  Projects: () => appBinding().Projects(),
  RefreshProjects: () => appBinding().RefreshProjects(),
  SearchProjects: (query) => appBinding().SearchProjects(query),
  Scan: () => appBinding().Scan(),
  Icon: (path) => appBinding().Icon(path),
  Launch: (req) => appBinding().Launch(req),
  SetLaunchSelection: (req) => appBinding().SetLaunchSelection(req),
  Debug: (stage, detail) => appBinding().Debug(stage, detail),
  ChooseFolder: () => appBinding().ChooseFolder(),
  ChooseWSLFolder: () => appBinding().ChooseWSLFolder(),
  WSLDistros: () => appBinding().WSLDistros(),
  ElevationStatus: () => appBinding().ElevationStatus(),
  SwitchElevation: (mode) => appBinding().SwitchElevation(mode),
  Hide: () => appBinding().Hide(),
  Show: () => appBinding().Show(),
  WindowMode: () => appBinding().WindowMode(),
  OpenSettingsWindow: () => appBinding().OpenSettingsWindow(),
  CloseSettingsWindow: () => appBinding().CloseSettingsWindow(),
};
