# 从 Swift 迁移到 MyGo：分析与代码导读

这份文档说明 StockFloat 从原生 Swift（AppKit + SwiftUI）迁移到 [MyGo](https://mygo.egoist.dev/docs) 时，每一块功能放到了哪里、为什么这样放，以及读这份代码的顺序。

## 一、MyGo 里和这个项目相关的部分

MyGo 用 Go 写桌面应用。一个窗口要么显示**网页**（系统自带的 WebView，macOS 上是 WKWebView），要么显示用 Go 画的**原生 UI**（`ui` 包）。本项目用的是网页窗口：`mygo init` 生成的就是网页模板（Go + Vite + TypeScript）。

用到的 MyGo 能力：

| 需要的功能 | MyGo 提供的 API | 文档 |
|---|---|---|
| 窗口：无边框、透明、毛玻璃、置顶、不透明度 | `mygo.NewWindow(WindowOptions{Frameless, Transparent, Vibrancy, AlwaysOnTop, Opacity})` | windows.md |
| 窗口在所有桌面、鼠标穿透 | `SetVisibleOnAllWorkspaces`、`SetIgnoreMouseEvents` | windows.md |
| 页面调用 Go | `mygo.Bind(service)` + 自动生成的 `src/mygo.ts` | bindings.md |
| Go 推数据给页面 | `mygo.NewEvent[T]("name")`，`Emit` / `Broadcast` | bindings.md |
| Go 的 error → 前端异常 | 方法返回 `error`，前端 `isCallError(err)` | bindings.md |
| 右键菜单、菜单栏、Dock 菜单 | `mygo.NewMenu`、`Menu.Popup`、`App.SetMenu`、`App.Dock.SetMenu` | menus.md |
| 菜单栏图标 | `mygo.NewTray`（模板图标、`PopUpMenu`） | menus.md |
| 全局快捷键 | `mygo.GlobalShortcut.Register("Ctrl+Alt+S", fn)` | native.md |
| Dock 图标开关 | `App.SetActivationPolicy(Regular / Accessory)` + `LSUIElement` | app.md、launchers.md |
| 开机启动 | `App.SetOpenAtLogin`、`App.OpenAtLogin` | app.md |
| 再次打开 App 时显示面板 | `App.OnActivate`、`App.RequestSingleInstanceLock`、`OnSecondInstance` | app.md |
| 显示器信息 | `mygo.Screen.Displays()`、`PrimaryDisplay().WorkArea` | native.md |
| 自动更新 | `mygo.Updater.Check` / `Update.Install` / `App.Relaunch` + `mygo keygen` | updates.md |
| 打包、签名、DMG | `mygo build`，配置在 `mygo.config.ts` | distribution.md、configuration.md |
| CI 发版 | GitHub Actions 工作流 | github-actions.md |

用不上的部分：documents、audio、剪贴板/拖放、SQLite、watch、terminal、glass、game 插件，以及 fetch / websocket 插件（行情请求放在 Go 里做，不需要从页面发网络请求）。

### 一个关键的版本问题

MyGo 最新发布版是 **v0.4.8**。文档网站跟随 `main` 分支，里面描述的 `WindowOptions.Panel`（Spotlight 那种悬浮面板：不激活 App、能盖在全屏应用上）**还没有进任何发布版**。原版面板的核心体验正是这个：

- 浮在所有窗口之上，所有桌面（Space）都在，全屏应用上面也在；
- **永远不抢键盘焦点、不激活 App**：看行情时不会打断正在写代码的编辑器。

v0.4.8 的普通窗口做不到第二点。所以 `native_darwin.go` 通过 MyGo 自己也在用的 purego（无 cgo 的 Objective-C 运行时调用），在 MyGo 创建好的 `NSWindow` 上做了几处调整。等 MyGo 发布了带 `Panel` 的版本，可以评估换成 `Panel: true` 并删掉大部分原生代码。不过 MyGo 的 Panel 显示时会拿到键盘焦点，和原版“永不拿焦点”不完全一样，换之前要先确认。

## 二、架构：Go 管数据，页面只负责画

```
                 ┌──────────────── Go（main 包 + internal/）────────────────┐
 腾讯 HTTP ──┐   │                                                          │
 Finnhub WS ─┼──▶│ feed.Provider ──emit──▶ Store（配置 + 行情 + 断线状态）    │
 longbridge ─┘   │                           │ 50ms 合并后回调                │
   serve         │                           ▼                              │
                 │ view.go: panelState() → PanelState（已本地化、已算好涨跌）│
                 │                           │ PanelChanged.Emit(panel, …)  │
                 │ menus.go / panel.go / native_darwin.go / updates.go      │
                 └───────────────────────────┼──────────────────────────────┘
                                             ▼
                 ┌──────── 页面（src/，WKWebView）─────────┐
                 │ panel.ts：按 symbol 原地更新行、闪烁、     │
                 │ 30 分钟过期变淡；ResizeObserver →        │
                 │ Panel.resize(w, h) 让窗口贴合内容         │
                 │ settings.ts：表单 → Settings.save(...)   │
                 └─────────────────────────────────────────┘
```

为什么这样分：

1. **行情逻辑留在 Go。** 三个数据源（GB18030 解码、WebSocket 心跳、子进程 JSON-RPC）在 Go 里写起来直接，也能像原版一样写单元测试。放进页面的话，要用 fetch / websocket 插件绕开 CORS，Longbridge 的子进程更是只能在 Go 里跑。
2. **所有文案在 Go（`internal/l10n`）。** 菜单是 Go 建的，所以中英文留在一处。页面需要的文字随 `PanelState` / `SettingsForm` 一起发过去。MyGo 0.5 才有 ICU 消息目录（i18n.md），0.4.8 用不了。
3. **页面只做布局。** `PanelState` 里的标签、涨跌幅、过期时间、断线提示都在 Go 里算好，页面拿到就画。

## 三、文件对照

| Swift 原版 | MyGo 版 | 说明 |
|---|---|---|
| `main.swift`、`AppDelegate.swift` | `main.go`、`app.go` | 生命周期、改设置后的联动（不透明度、穿透、Dock、语言） |
| `FloatingPanel.swift` | `panel.go` + `native_darwin.go` | 放置、跟随内容改尺寸、记住位置；不拿焦点、拖动、右键、圆角毛玻璃 |
| `Config.swift` | `internal/config` | **同一个文件** `~/.config/stock-float/config.json`、同样的字段和默认值 |
| `Quote.swift` | `internal/market` | 代码解析规则、涨跌幅、30 分钟过期 |
| `QuoteStore.swift` | `store.go` | 用代号（generation）丢弃已停止数据源的迟到事件；50ms 合并推送 |
| `Providers/*.swift` | `internal/feed/*.go` | `AsyncStream` → `Run(ctx, symbols, emit)`；`Task` 取消 → `context` |
| `Localization.swift` | `internal/l10n` | 文案逐字照搬 |
| `Hotkey.swift`（Carbon） | `hotkey.go` + `mygo.GlobalShortcut` | 只剩把 `ctrl+opt+s` 转成 `Ctrl+Alt+S` |
| `Updater.swift`（自写 zip 更新） | `updates.go` + `mygo.Updater` | 改用 MyGo 的签名更新（Ed25519、增量更新），界面保持原样 |
| `QuoteListView.swift` | `index.html`、`src/panel.ts`、`src/panel.css` | |
| `SettingsView.swift` | `settings.html`、`src/settings.ts`、`src/settings.css` | |
| `scripts/bundle.sh`、`release.sh` | `mygo build`、`mygo.config.ts` | Info.plist、签名、DMG 都由 CLI 完成 |
| `scripts/make-icon.swift` | `scripts/make-icons.swift` | 另外画出菜单栏图标和盘前/盘后/夜盘图标（SF Symbols → PNG） |
| `Tests/*` | `*_test.go` | 原有测试全部移植，包括用 shell 脚本假冒 longbridge CLI 的进程测试 |

## 四、功能清单

| 功能 | 实现位置 |
|---|---|
| 悬浮在所有窗口和桌面之上，包括全屏应用 | `native_darwin.go` `makePanel`：collectionBehavior = CanJoinAllSpaces \| Stationary \| IgnoresCycle \| FullScreenAuxiliary，floating 层级 |
| 永不拿焦点、点击不激活 App | 把窗口类换成子类 `StockFloatPanel`（`canBecomeKeyWindow` 返回 NO），并调用 `_setPreventsActivation:` |
| 任意位置拖动、右键菜单 | 子类重写 `sendEvent:`，左键按下调用 `performWindowDragWithEvent:`，右键按下打开 `App.menu()` |
| 圆角 HUD 毛玻璃 | `Vibrancy: VibrancyHUD` + 给 `NSVisualEffectView` 设置可拉伸的圆角 `maskImage`（和原版同一招） |
| 窗口贴合内容、左上角不动 | 页面 `ResizeObserver` → `Panel.resize` → `SetBounds` |
| 首次放在右上角，之后记住位置 | `initialPosition` / `savePosition`，存在 `~/Library/Application Support/StockFloat/panel.json` |
| 鼠标穿透 + 按住 ⌥ 临时操作 | `applyClickThrough`：每 150ms 读 `CGEventSourceFlagsState`，不需要辅助功能权限 |
| 三个数据源、断线提示 | `internal/feed`、`view.go` |
| 盘前/盘后/夜盘第二行 | `LongbridgeBook` 合并逻辑和原版一致 |
| 价格变动闪烁（只闪常规时段） | `panel.ts` 的 `flash`：行元素按 symbol 复用，动画不会被下一次推送打断 |
| 30 分钟不更新变淡 | Go 算出 `staleAt`，页面每分钟检查一次 |
| 外观（配色/字号/不透明度）、语言 | `menus.go` → `App.change` |
| 菜单栏图标，每次打开时现建菜单 | `ready()`：`OnClick` 里先 `SetMenu`，再 `PopUpMenu`，然后 `SetMenu(nil)` |
| Dock 图标开关、Dock 菜单 | `LSUIElement` + `SetActivationPolicy`、`dockMenu()` |
| 全局快捷键 ⌃⌥S | `registerHotkey` |
| 开机启动 | `loginItem` |
| 设置窗口（校验代码，无法识别的标红） | `Settings.Save` 返回 `error`，页面用 `isCallError` 显示 |
| 应用内更新 | `updates.go`，需要先配置密钥（见第六节） |

## 五、建议的阅读顺序

1. **`internal/market`、`internal/config`**：纯 Go 的领域模型，对照 `Quote.swift`、`Config.swift` 看。
2. **`internal/feed/tencent.go`**：最简单的数据源，看 `Provider` 接口怎么取代 `AsyncStream`。
3. **`internal/feed/finnhub.go`、`longbridge.go`**：goroutine、`context` 取消、子进程管道；配合对应的 `_test.go`。
4. **`store.go`**：并发安全的状态，以及从数据源到页面的那一跳。
5. **`main.go` → `app.go` → `services.go`**：MyGo 的入口，`Bind` 和 `NewEvent`；然后看自动生成的 `src/mygo.ts`，体会 Go 结构体和文档注释怎样变成 TypeScript 类型。
6. **`src/panel.ts`**：页面这一侧。
7. **`panel.go`、`menus.go`**：窗口和菜单 API。
8. **`native_darwin.go`**：最后读。它展示 MyGo 没覆盖到的地方怎么通过 `NativeHandle()` 和 purego 直接调用 AppKit。

MyGo 的几条线程规则，读代码时会反复碰到（app.md「Threads」）：

- 事件监听、菜单点击、快捷键回调都跑在**主线程**，要短；
- 被绑定的方法（`Panel.Resize` 等）每次调用跑在**各自的 goroutine** 上，可以阻塞；
- 所有 MyGo API 在任何 goroutine 都能安全调用；`mygo.RunOnMain` 把一段代码放到主线程执行。`App` 里标了 `// main thread` 的字段只在主线程读写。

## 六、开发、构建、发布

```sh
bun run dev      # 开发版 "StockFloat Dev"：改 Go 会重启，改前端走 Vite 热更新
bun run test     # go test ./... + tsc
bun run build    # build/darwin-arm64/StockFloat.app + DMG
```

**启用应用内更新**（只需做一次）：

1. 运行 `bunx mygo keygen`，生成 `mygo-update.key`（私钥）和 `mygo-update.pub`（公钥）；
2. 把公钥填进 `mygo.config.ts` 的 `updates.publicKey`，取消 `github: "lesenelir/stock-float-mygo"` 那行的注释；
3. 运行 `gh secret set MYGO_UPDATER_PRIVATE_KEY < 私钥路径`；
4. 改 `package.json` 的 `version`，推送 `v<version>` 标签。`release.yml` 会构建 universal 包并上传到草稿 Release，检查后再发布。

没配置之前，App 照常运行，只是不检查更新（`mygo.Updater.Enabled()` 为 false）。

## 七、和原版的差异、已知限制

- **体积和内存。** 原版是 1.6 MB 的 App、约 17 MB 内存；这一版 App 约 11 MB（DMG 3.4 MB），开发版进程约 100 MB RSS，WKWebView 的辅助进程另算。这是网页窗口的固定开销。想更省，可以把面板改成 MyGo 原生 UI（`Content: ui.View(...)`），不起 WebView 进程。
- **私有 API。** `_setPreventsActivation:` 是私有方法，Electron 和 tauri-nspanel 也在用。如果哪天这个方法没了，代码会跳过它：面板仍然不会拿键盘焦点，只是点击时 App 会被激活。
- **Dock 图标开启时盖在全屏应用上。** macOS 只允许 NSPanel 或后台（accessory）App 的窗口出现在别的 App 的全屏空间里。这里的窗口是 NSWindow，所以关闭 Dock 图标时一定可以；开启时取决于 `_setPreventsActivation:` 的效果，需要实测。
- **生成的 TS 类型。** `mygo generate` 只在有绑定方法的包（main）里找常量，所以 `internal/` 里的 `Session`、`UpColor` 在 TypeScript 里是 `string`，不是联合类型。
- **App 名称与标识符。** 名称同为 StockFloat，标识符是 `com.lesenelir.stockfloat.mygo`，两个版本可以同时安装，开机启动和单实例锁互不影响，并且共用同一份配置文件。
- **Linux / Windows。** 能编译（`native_other.go`），面板会置顶、在所有桌面显示，页面自己处理拖动和右键；但会拿焦点，也没有 ⌥ 穿透。
