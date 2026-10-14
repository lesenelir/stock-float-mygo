import { defineConfig } from "mygo-cli";
import pkg from "./package.json" with { type: "json" };

export default defineConfig({
  name: "StockFloat",
  // Apart from the Swift app's com.lesenelir.stockfloat, so the two can be
  // installed side by side without sharing login items or the single
  // instance lock. They share ~/.config/stock-float/config.json.
  identifier: "com.lesenelir.stockfloat.mygo",
  // The release workflow checks that the tag matches it.
  version: pkg.version,
  copyright: "© 2026 lesenelir",

  // `mygo dev` runs devCommand and loads devUrl; `mygo build` runs
  // buildCommand and embeds frontendDist into the app.
  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",

  macos: {
    infoPlist: {
      // A background app from launch, so a user who turned the Dock icon off
      // never sees it flash; main.go brings the Dock icon back when it is on.
      LSUIElement: true,
      LSApplicationCategoryType: "public.app-category.finance",
    },
  },

  // Signed auto-updates from GitHub releases (see docs/migration.zh-CN.md). Create
  // the key pair once with `bunx mygo keygen`, paste the public key here, and
  // keep the private key in the MYGO_UPDATER_PRIVATE_KEY secret of the
  // repository. Until then builds do not update themselves.
  // updates: {
  //   publicKey: "…the contents of mygo-update.pub…",
  //   github: "lesenelir/stock-float-mygo",
  // },
});
