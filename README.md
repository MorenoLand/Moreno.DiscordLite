# Discord Wails

An unofficial Discord desktop wrapper built with Wails 3. It loads Discord at `https://discord.com/app`, injects the bundled Vencord browser build, provides the custom frameless title bar and tray controls, and exposes the local Discord RPC pipe.

## Build

Requirements: Go, Node.js, pnpm, Wails 3, and UPX on Windows.

```sh
pnpm install
pnpm build
```

The executable is written to `bin/discord.exe` on Windows, `bin/discord` on Linux, and `bin/discord` on macOS. Windows release builds are stripped and compressed with UPX.

The Vencord browser assets are refreshed from the upstream devbuild at startup when the local update marker is older than 24 hours. The application is an unofficial wrapper and is not affiliated with Discord or Vencord.

## License

MIT. See [LICENSE](LICENSE).
