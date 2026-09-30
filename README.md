# Unofficial Discord desktop wrapper

An unofficial Discord desktop wrapper built with Wails 3. It loads Discord at `https://discord.com/app`, downloads and injects the Vencord browser build, provides the custom frameless title bar and tray controls, and exposes the local Discord RPC pipe on Windows.

## Features

- Discord web app hosted in a native Wails window.
- Minimize, maximize, close, window dragging, and tray show/hide/quit controls.
- F12 DevTools access for inspecting the live Discord DOM and injection point.
- Vencord browser JavaScript and CSS downloaded from upstream and cached locally.
- Vencord devbuild refresh at startup when either cached file is missing or older than 24 hours.
- Discord IPC named-pipe compatibility for the supported Windows RPC commands.
- System handling for external links and Discord attachment downloads.

## Requirements

- Go 1.25 or newer.
- Node.js and pnpm 10.13.1.
- Wails 3 CLI matching the pinned `github.com/wailsapp/wails/v3` dependency (`v3.0.0-beta.11`).
- UPX on Windows; the Windows package task uses `--best --lzma` compression.
- A working WebView runtime supplied by the target operating system.

## Build

Install the locked frontend dependencies and package the application:

```sh
pnpm install --frozen-lockfile
pnpm build
```

The production build runs the frontend build, generates the Windows application resources, builds with stripped release flags, and compresses the Windows executable with UPX. The output is written to `bin/discord.exe` on Windows and `bin/discord` on Linux and macOS.

For development:

```sh
pnpm dev
```

The application needs network access to load Discord. Press F12 in a running build to open DevTools. The app is not signed or affiliated with Discord, Vencord, or their respective owners.

## Runtime assets

The application downloads `browser.js` and `browser.css` from the Vencord devbuild release and caches them in the user's OS cache directory. It refreshes the cache at startup if either file is missing or older than 24 hours. If the download fails and there is no cached copy, Discord loads without Vencord. The upstream assets are not checked into this repository or embedded in the executable.

This wrapper executes remote Discord content and third-party Vencord code inside the application. Use only builds and upstream assets you trust, and review [SECURITY.md](SECURITY.md) before distributing a build.

## Development checks

```sh
go test ./...
go vet ./...
git diff --check
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the change and pull-request requirements.

## Licensing

The original project source is MIT-licensed; see [LICENSE](LICENSE) and [LICENSE.md](LICENSE.md). Vencord is downloaded at runtime and retains its upstream terms; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). Discord content, trademarks, and services remain under Discord's terms.
