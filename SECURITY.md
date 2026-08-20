# Security policy

## Scope

Moreno.DiscordLite is a desktop wrapper that loads remote Discord content, executes Vencord browser assets, writes downloads selected by the user, and exposes a Windows Discord IPC named pipe. Treat builds and upstream browser assets as security-sensitive inputs.

## Reporting a vulnerability

Do not disclose an unpatched vulnerability in a public issue, pull request, chat, or screenshot. Use GitHub's private vulnerability reporting for this repository:

[Report a vulnerability](https://github.com/MorenoLand/Moreno.DiscordLite/security/advisories/new)

Include the affected commit or build, operating system, reproduction steps, impact, relevant logs, and any mitigation you have identified. Remove Discord tokens, cookies, authorization headers, RPC payloads, personal data, and private URLs before submitting.

If private reporting is unavailable, open an issue containing only a request for a private security contact. Do not include vulnerability details until a private channel is established.

## Upstream reports

Report vulnerabilities in Discord itself to Discord. Report vulnerabilities in Vencord or its devbuild artifacts to [Vencord](https://github.com/Vendicated/Vencord/security). Include enough information to distinguish an upstream issue from wrapper behavior.

## Handling

Reports are reviewed on a best-effort basis. Please allow maintainers time to validate the report and prepare a fix or mitigation before public disclosure. There is no guaranteed response or remediation time.
