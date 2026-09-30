# Third-party notices

## Vendicated/Vencord

The application downloads these Vencord browser artifacts at runtime and caches them outside the repository:

- `browser.js`
- `browser.css`

The devbuild endpoints used by `main.go` are:

- <https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.js>
- <https://github.com/Vendicated/Vencord/releases/download/devbuild/browser.css>

Vencord is distributed by its upstream project under the GNU General Public License, version 3. Review the upstream [Vencord license](https://github.com/Vendicated/Vencord/blob/main/LICENSE) for the applicable terms. This repository's MIT license does not relicense these runtime downloads.

## Wails and Go dependencies

Wails and the Go modules listed in `go.mod` remain under their respective upstream licenses. Consult each upstream project for its license and notices before redistributing a modified dependency set.

## Discord

Discord content is loaded from Discord at runtime. Discord names, logos, content, trademarks, and services are owned by their respective holders. Moreno.DiscordLite is an unofficial wrapper and is not affiliated with or endorsed by Discord.
