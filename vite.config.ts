import { defineConfig } from "vite";

export default defineConfig({
  server: {
    // The app loads this address during `mygo dev` (devUrl in mygo.config.ts).
    port: 5173,
    strictPort: true,
    // The development app and the packaged builds are not the page's.
    watch: { ignored: ["**/.mygo/**", "**/build/**"] },
  },
  build: {
    // Two pages: the panel (/) and the settings window (/settings.html).
    rolldownOptions: {
      input: { panel: "index.html", settings: "settings.html" },
    },
  },
});
