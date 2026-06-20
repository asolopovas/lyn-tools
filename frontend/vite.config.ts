import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig, type Plugin } from "vite";

function keepDistTracked(): Plugin {
  let outDir = "dist";
  return {
    name: "keep-dist-tracked",
    apply: "build",
    configResolved(config) {
      outDir = resolve(config.root, config.build.outDir);
    },
    closeBundle() {
      writeFileSync(resolve(outDir, ".gitkeep"), "");
    },
  };
}

export default defineConfig({
  plugins: [vue(), tailwindcss(), keepDistTracked()],
  server: {
    host: "127.0.0.1",
    port: 5173,
    strictPort: true,
  },
});
