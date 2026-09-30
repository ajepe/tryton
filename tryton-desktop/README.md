# Tryton Desktop (Wails)

Desktop client wrapping the Talon web client in a native window.

## Prerequisites

- Go 1.25+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- Web client build output from `../sao-ts`

## Build

```bash
# build the web client first
(cd ../sao-ts && npm install && npm run build)
cp -r ../sao-ts/dist/* frontend/dist/

wails build
```

The resulting binary is in `build/bin/`.

## Features

- Native file dialogs for import/export (`OpenFile`, `SaveFile`)
- Native notifications and window-title badge (`Notify`)
- Same JSON-RPC/session/bus behavior as the web client because it embeds the
  same SPA and serves it from the Wails asset server.

## Targets

Wails produces `.exe`/`.msi` on Windows, `.app`/`.dmg` on macOS and
`.AppImage`/`.deb` on Linux with the matching platform toolchains.
