import type { WailsApp } from "./types";

declare global {
  interface Window {
    go?: {
      main?: { App?: WailsApp };
      lyn?: { App?: WailsApp };
    };
  }
}
