# Frontend

Part of the [architecture map](../ARCHITECTURE.md). Rules for writing frontend code live in the [quality policy](QUALITY.md).

- [`frontend/src/App.vue`](../frontend/src/App.vue) coordinates launcher UI state and Wails events. It sends query text to backend search and renders matches.
- Components render panels. Small modules own backend access, cache, themes, icons, types, hotkeys, and launch actions.
- Keyboard mappings live in [`frontend/src/hotkeys.ts`](../frontend/src/hotkeys.ts), reused by input and window handlers.
- Local storage is only a warm UI cache, never search or ranking truth.
- Launcher panel height is `min(<configured>px, 100vh)` with `width: 100%`, so it never exceeds the live viewport. The results list owns its scroll (`overflow-y: auto`).
- The `100vh` cap is required: under fractional display scale WebKitGTK locks the viewport to creation-size, and `DisableResize` keeps `WindowSetSize` from changing it.
- Do not use `overflow-y: overlay`: WebKitGTK treats it as `visible` (no scrolling).
