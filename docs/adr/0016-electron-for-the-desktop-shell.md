# The desktop app is Electron, not Tauri

Tauri is the obvious pick on bundle size, and it was rejected anyway: it
renders through the host's WebView, which on Linux means WebKitGTK and a
visibly different — sometimes broken — result from the browser the web app is
developed against. Electron ships its own Chromium, so the desktop shell
renders exactly what the browser does on every platform. The desktop app is a
thin shell around the same web app, so a heavier runtime buys consistency at
no development cost.

Decided: 2026-07-25
