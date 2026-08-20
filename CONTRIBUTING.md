# Contributing

Contributions are welcome when they keep the wrapper reliable and preserve the behavior of the previous desktop implementation where parity is intended.

## Before you start

Install Go 1.25 or newer, Node.js, pnpm 10.13.1, the Wails 3 CLI matching `v3.0.0-beta.11`, and UPX on Windows. Clone the repository, then install the locked frontend dependencies:

```sh
pnpm install --frozen-lockfile
```

## Local workflow

Use the development task while working on the UI or injection behavior:

```sh
pnpm dev
```

Run the checks that apply to your change before opening a pull request:

```sh
go test ./...
go vet ./...
pnpm build
git diff --check
```

On Windows, verify the packaged executable with `upx -t bin/discord.exe` when the build includes packaging changes.

## Change guidelines

- Keep changes focused and preserve the existing Discord, Vencord, RPC, tray, download, and title-bar behavior unless the change explicitly targets one of them.
- Put application logic in source files and build configuration, not generated output. Do not commit `bin/`, `*.syso`, `frontend/dist/`, or dependency directories.
- Review changes to `vencord/browser.js` and `vencord/browser.css`; they can be refreshed by running the application and remain third-party assets.
- Do not commit Discord tokens, cookies, RPC payloads, credentials, private URLs, signing keys, or machine-specific paths.
- For UI or injection changes, include the affected route/state and a screenshot or DevTools observation when it helps reviewers reproduce the behavior.
- Update the relevant documentation when build, runtime, security, or contribution behavior changes.

## Pull requests

Describe the problem, the user-visible change, and the validation performed. Keep unrelated formatting or generated-file churn out of the pull request. A maintainer may request a smaller change or an additional parity check before merging.

By submitting a contribution, you agree that the original work you contribute may be distributed under the MIT License in [LICENSE](LICENSE). Third-party material remains under its upstream terms as described in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
