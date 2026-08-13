# Frontend

Part of the [architecture map](../ARCHITECTURE.md). Rules for writing frontend code live in the [quality policy](QUALITY.md).

- [`frontend/src/App.vue`](../frontend/src/App.vue) composes launcher state and renders panels. It sends query text to backend search and does not own native-window or global-input behavior.
- [`frontend/src/themeState.ts`](../frontend/src/themeState.ts) owns derived theme colors, opacity, and root CSS-variable synchronization.
- [`frontend/src/appWindow.ts`](../frontend/src/appWindow.ts) owns Wails window events, sizing, focus/blur behavior, settings visibility, native-selection polling, ready acknowledgements, and cleanup.
- [`frontend/src/launcherKeyboard.ts`](../frontend/src/launcherKeyboard.ts) owns global keyboard registration and action dispatch.
- Components render panels. Small modules own backend access, cache, themes, icons, types, settings, launcher state, and launch actions.
- The UI's token system, type scale, motion, and `ui/*` component idioms are captured in [`DESIGN-LANGUAGE.md`](DESIGN-LANGUAGE.md) for reuse outside the Vue app (e.g. Claude Design).
- Keyboard mappings live in [`frontend/src/hotkeys.ts`](../frontend/src/hotkeys.ts), reused by input and window handlers.
- Local storage is only a warm UI cache, never search or ranking truth.
- Launcher panel height is `min(<configured>px, 100vh)` with `width: 100%`, so it never exceeds the live viewport. The results list owns its scroll (`overflow-y: auto`).
- The `100vh` cap is required: under fractional display scale WebKitGTK locks the viewport to creation-size, and `DisableResize` keeps `WindowSetSize` from changing it.
- Do not use `overflow-y: overlay`: WebKitGTK treats it as `visible` (no scrolling).
