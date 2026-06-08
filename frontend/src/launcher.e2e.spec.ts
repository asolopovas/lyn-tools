import { expect, test, type Page } from "@playwright/test";
import path from "node:path";

type LaunchCall = { path: string; action: string };
type DebugCall = { stage: string; detail: string };

const projectPath = process.env.LYN_E2E_PROJECT_PATH ?? path.resolve("..");
const appData = process.env.APPDATA ?? path.join(path.dirname(projectPath), "AppData", "Roaming");
const localAppData =
  process.env.LOCALAPPDATA ?? path.join(path.dirname(projectPath), "AppData", "Local");

const project = {
  name: path.basename(projectPath),
  path: projectPath,
  kind: "go",
  detectedAt: "2026-01-01T00:00:00Z",
  usageCount: 0,
  lastLaunchedAt: "",
};
const secondProject = {
  ...project,
  name: "zz-second-project",
  path: path.join(projectPath, "zz-second-project"),
};
const scrollProjects = Array.from({ length: 12 }, (_, index) => ({
  ...project,
  name: `lyn-scroll-project-${index + 3}`,
  path: path.join(projectPath, `lyn-scroll-project-${index + 3}`),
}));
const projects = [project, secondProject, ...scrollProjects];
const config = {
  path: path.join(appData, "lyn", "lyn.json"),
  cache: { dir: path.join(localAppData, "lyn") },
  startup: { enabled: true, startHidden: true },
  scanner: { roots: [], maxDepth: 4, concurrency: 4, timeout: "5s", watch: false },
  hotkey: { binding: "Win+D" },
  ui: {
    theme: "power-run",
    backgroundOpacity: 0.98,
    selectionColor: "#333333",
    windowPlacement: "center",
    clearQueryOnShow: true,
    workspaceQueryShortcut: "{",
  },
};

async function installWailsMocks(page: Page): Promise<void> {
  await page.addInitScript(
    ({ config, projects }) => {
      const state = {
        launches: [] as LaunchCall[],
        debugs: [] as DebugCall[],
        hides: [] as string[],
        listeners: new Map<string, Array<(...data: unknown[]) => void>>(),
        refreshes: 0,
        refreshResolvers: [] as Array<() => void>,
      };
      Object.assign(window, {
        __lynLaunches: state.launches,
        __lynDebugs: state.debugs,
        __lynHides: state.hides,
        __lynDeferRefresh: false,
        __lynPendingRefreshCount: () => state.refreshResolvers.length,
        __lynRefreshCount: () => state.refreshes,
        __lynResolveRefreshes: () => {
          for (const resolve of state.refreshResolvers.splice(0)) resolve();
        },
        runtime: {
          EventsEmit: (name: string, ...data: unknown[]) => {
            for (const listener of state.listeners.get(name) ?? []) listener(...data);
          },
          EventsOnMultiple: (
            name: string,
            callback: (...data: unknown[]) => void,
            maxCallbacks: number,
          ) => {
            let remaining = maxCallbacks;
            const wrapped = (...data: unknown[]) => {
              if (remaining === 0) {
                return;
              }
              if (remaining > 0) {
                remaining -= 1;
              }
              callback(...data);
            };
            const listeners = state.listeners.get(name) ?? [];
            listeners.push(wrapped);
            state.listeners.set(name, listeners);
            return () =>
              state.listeners.set(
                name,
                listeners.filter((item) => item !== wrapped),
              );
          },
          EventsOn: (name: string, callback: (...data: unknown[]) => void) =>
            window.runtime.EventsOnMultiple(name, callback, -1),
          WindowCenter: () => {},
          WindowSetPosition: () => {},
          WindowSetSize: () => {},
        },
        go: {
          main: {
            App: {
              Config: async () => config,
              SaveConfig: async (next: unknown) => next,
              Projects: async () => projects,
              RefreshProjects: async () => {
                if (window.__lynDeferRefresh) {
                  await new Promise<void>((resolve) => state.refreshResolvers.push(resolve));
                }
                state.refreshes += 1;
                return projects;
              },
              SearchProjects: async (query: string) =>
                projects
                  .filter((item) =>
                    `${item.name} ${item.kind} ${item.path}`
                      .toLowerCase()
                      .includes(query.trim().toLowerCase()),
                  )
                  .slice(0, 12),
              Scan: async () => ({ count: 1 }),
              Icon: async () => "",
              Launch: async (request: LaunchCall) => {
                state.launches.push(request);
                return { command: "code", args: [request.path] };
              },
              Debug: async (stage: string, detail: string) => {
                state.debugs.push({ stage, detail });
              },
              SetLaunchSelection: async () => {},
              ChooseFolder: async () => "",
              Hide: async () => {
                state.hides.push("hide");
              },
              WindowMode: async () => "launcher",
              OpenSettingsWindow: async () => {
                state.hides.push("settings");
              },
              CloseSettingsWindow: async () => {},
            },
          },
        },
      });
    },
    { config, projects },
  );
}

declare global {
  interface Window {
    __lynLaunches: LaunchCall[];
    __lynDebugs: DebugCall[];
    __lynHides: string[];
    __lynDeferRefresh: boolean;
    __lynPendingRefreshCount: () => number;
    __lynRefreshCount: () => number;
    __lynResolveRefreshes: () => void;
    runtime: {
      EventsEmit: (name: string, ...data: unknown[]) => void;
      EventsOn: (name: string, callback: (...data: unknown[]) => void) => () => void;
      EventsOnMultiple: (
        name: string,
        callback: (...data: unknown[]) => void,
        maxCallbacks: number,
      ) => () => void;
    };
  }
}

async function fillQuery(page: Page, text = "lyn"): Promise<void> {
  await page.getByPlaceholder("Start typing...").fill(text);
}

async function expectLaunches(page: Page, launches: LaunchCall[]): Promise<void> {
  await expect.poll(() => page.evaluate(() => window.__lynLaunches)).toEqual(launches);
}

async function waitForProject(page: Page, name: string): Promise<void> {
  await page.getByText(name, { exact: true }).waitFor();
}

function selectedProjectName(page: Page) {
  return page.locator("li.selected strong");
}

async function expectSelectedProject(page: Page, name: string): Promise<void> {
  await expect(selectedProjectName(page)).toHaveText(name);
}

async function pressKey(page: Page, key: string, count = 1): Promise<void> {
  for (let index = 0; index < count; index += 1) {
    await page.keyboard.press(key);
  }
}

test.beforeEach(async ({ page }) => {
  await installWailsMocks(page);
  await page.goto("/");
});

test("Enter in the launcher input logs and launches selected project in Code", async ({ page }) => {
  await fillQuery(page);
  await page.keyboard.press("Enter");
  await expectLaunches(page, [{ path: project.path, action: "code" }]);
  await expect
    .poll(() => page.evaluate(() => window.__lynDebugs.map((entry) => entry.stage)))
    .toContain("launch.request");
});

test("mouse launching selected row reaches the backend launch binding", async ({ page }) => {
  await fillQuery(page);
  await page.getByText(project.name, { exact: true }).click();
  await expectLaunches(page, [{ path: project.path, action: "code" }]);
});

test("result action buttons launch their action instead of the row default", async ({ page }) => {
  await fillQuery(page);
  await page.getByTitle("Open containing folder (Ctrl+Shift+E)").first().click();
  await expectLaunches(page, [{ path: project.path, action: "reveal" }]);
});

test("result action keyboard shortcut launches the selected action", async ({ page }) => {
  await fillQuery(page);
  await page.keyboard.press("Control+Shift+E");
  await expectLaunches(page, [{ path: project.path, action: "reveal" }]);
});

test("Alt number launches the matching visible result", async ({ page }) => {
  await waitForProject(page, secondProject.name);
  await page.keyboard.press("Alt+2");
  await expectLaunches(page, [{ path: secondProject.path, action: "code" }]);
});

test("live refresh after launcher show preserves arrow selection", async ({ page }) => {
  await waitForProject(page, project.name);
  await page.evaluate(() => {
    window.__lynDeferRefresh = true;
    window.runtime.EventsEmit("launcher-shown", 1);
  });
  await pressKey(page, "ArrowDown");
  await expectSelectedProject(page, secondProject.name);
  await expect
    .poll(() => page.evaluate(() => window.__lynPendingRefreshCount()))
    .toBeGreaterThan(0);
  await page.evaluate(() => {
    window.__lynResolveRefreshes();
    window.__lynDeferRefresh = false;
  });
  await expect.poll(() => page.evaluate(() => window.__lynRefreshCount())).toBeGreaterThan(0);
  await expectSelectedProject(page, secondProject.name);
});

test("arrow navigation scrolls the selected result into view", async ({ page }) => {
  await waitForProject(page, project.name);
  const list = page.locator("ul");
  await expect.poll(() => list.evaluate((element) => element.scrollTop)).toBe(0);
  await pressKey(page, "ArrowDown", 8);
  await expect.poll(() => list.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
  await expect
    .poll(() =>
      page.locator("li.selected").evaluate((selected) => {
        const listElement = selected.closest("ul");
        if (!listElement) {
          return false;
        }
        const selectedRect = selected.getBoundingClientRect();
        const listRect = listElement.getBoundingClientRect();
        return selectedRect.top >= listRect.top - 1 && selectedRect.bottom <= listRect.bottom + 1;
      }),
    )
    .toBe(true);
});

test("arrow navigation stops at the first and last results", async ({ page }) => {
  await waitForProject(page, project.name);
  await pressKey(page, "ArrowUp");
  await expectSelectedProject(page, project.name);
  await pressKey(page, "ArrowDown", 20);
  await expectSelectedProject(page, "lyn-scroll-project-12");
  await pressKey(page, "ArrowDown");
  await expectSelectedProject(page, "lyn-scroll-project-12");
  await pressKey(page, "ArrowUp");
  await expectSelectedProject(page, "lyn-scroll-project-11");
});

test("stationary mouse hover does not steal keyboard selection while scrolling up", async ({
  page,
}) => {
  await waitForProject(page, project.name);
  await pressKey(page, "ArrowDown", 20);
  await expectSelectedProject(page, "lyn-scroll-project-12");
  await page.locator("li").first().dispatchEvent("mouseenter");
  await pressKey(page, "ArrowUp");
  await expectSelectedProject(page, "lyn-scroll-project-11");
});

test("settings button opens the separate settings window without replacing the launcher", async ({
  page,
}) => {
  await page.getByTitle("Settings").click();
  await expect.poll(() => page.evaluate(() => window.__lynHides)).toEqual(["settings"]);
  await expect(page.getByPlaceholder("Start typing...")).toBeVisible();
});

test("Escape hides the launcher from the input", async ({ page }) => {
  await page.getByPlaceholder("Start typing...").focus();
  await page.keyboard.press("Escape");
  await expect.poll(() => page.evaluate(() => window.__lynHides)).toEqual(["hide"]);
});
