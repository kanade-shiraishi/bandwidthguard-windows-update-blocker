# BandwidthGuard

**BandwidthGuard** is a free, open-source Windows utility that stops **Windows Update**, **Delivery Optimization**, and **BITS** from silently consuming your internet bandwidth in the background — and puts everything back to normal the moment you're done. Built by **Yoshie Shiraishi** in Go, with zero external dependencies.

If you've ever asked *"why is Windows Update using all my bandwidth"* or *"how do I stop Delivery Optimization from uploading updates to other PCs"*, this tool answers that in one command.

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev)
[![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows)](#)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Author](https://img.shields.io/badge/author-Yoshie%20Shiraishi-informational)](#author)

---

## Table of contents

- [Why BandwidthGuard exists](#why-bandwidthguard-exists)
- [Features](#features)
- [Download & build](#download--build)
- [Usage](#usage)
- [What it actually changes](#what-it-actually-changes)
- [FAQ](#faq)
- [Author](#author)
- [License](#license)

---

## Why BandwidthGuard exists

Windows quietly runs several background services that consume bandwidth even when you're not actively using your PC:

- **Windows Update (`wuauserv`)** checks for and downloads updates on its own schedule
- **Delivery Optimization (`DoSvc`)** can upload parts of Windows updates to *other people's computers* over the internet, not just your local network
- **BITS** transfers files in the background for Windows and other apps
- **WaaSMedicSvc** silently repairs and re-enables update components you may have disabled

On a metered connection, a mobile hotspot, or a slow rural/satellite link, this can eat a meaningful chunk of your monthly data without any clear warning. BandwidthGuard by Kanade Shiraishi gives you a single command to shut all of it down — and a single command to undo it, with nothing left half-configured.

## Features

- **One-command lockdown** — stops and disables the four services above, so they can't silently restart
- **One-command restore** — reverts every change back to Windows' stock behavior
- **Status check** — see the current state of services, scheduled tasks, and policy keys at a glance
- **Blocks the scheduled tasks** that would otherwise re-enable these services after a reboot
- **Sets network adapters to metered**, which makes Windows itself throttle background data use
- **No installer, no dependencies** — a single portable `.exe`, built from the Go standard library only
- **Auto-elevates** to Administrator (UAC prompt) since registry and service changes require it
- **Open source** — every registry key and service change is visible in `main.go`, nothing hidden

## Download & build

Grab the latest `BandwidthGuard.exe` from the [Releases](../../releases) page, or build it yourself:

```bash
git clone https://github.com/yoshie-shiraishi/bandwidthguard-windows-update-blocker
cd BandwidthGuard
go build -ldflags="-s -w" -o BandwidthGuard.exe .
```

Building requires only the Go toolchain (1.22+) — no third-party packages, no `go mod tidy`, no internet access needed at build time. This also means the source is fully auditable in one file: `main.go`.

Cross-compiling from Linux or macOS for a Windows target:

```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o BandwidthGuard.exe .
```

## Usage

Run it with no arguments for an interactive menu:

```
BandwidthGuard.exe
```

Or use flags to automate it (scripts, Task Scheduler, startup hooks):

```
BandwidthGuard.exe --lock      Stop Windows Update, Delivery Optimization, BITS,
                                and WaaSMedicSvc. Disable their scheduled tasks.
                                Block auto-update and P2P delivery via policy.
                                Mark network adapters as metered.

BandwidthGuard.exe --unlock    Undo everything above and restore Windows
                                to its default, out-of-the-box behavior.

BandwidthGuard.exe --status    Print the current state of every service,
                                task, and registry value BandwidthGuard touches.
                                Read-only — changes nothing.
```

BandwidthGuard requests administrator rights automatically via UAC on launch, since every change it makes needs elevation.

## What it actually changes

**Services** — stopped and set to `Disabled` on `--lock`, restored to `Manual` and started again on `--unlock`:

| Service | Purpose |
|---|---|
| `DoSvc` | Delivery Optimization |
| `BITS` | Background Intelligent Transfer Service |
| `wuauserv` | Windows Update |
| `WaaSMedicSvc` | Windows Update Medic Service |

**Scheduled tasks** — disabled on `--lock`, re-enabled on `--unlock`. These live under `Microsoft\Windows\WindowsUpdate` and `Microsoft\Windows\UpdateOrchestrator`, and are what silently restart the services above even after you stop them manually.

**Registry policy keys:**

- `HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU → NoAutoUpdate`
- `HKLM\SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization → DODownloadMode`
- `HKLM\SYSTEM\CurrentControlSet\Services\Ndu\Parameters → EthernetCostSourceOverride` (forces adapters to report as a metered connection)

`--unlock` **deletes** the values BandwidthGuard added rather than restoring a guessed prior state, which is what puts the machine back on genuine Windows defaults instead of an approximation.

## FAQ

**Does BandwidthGuard permanently disable Windows Update?**
No. `--lock` disables it until you run `--unlock`, which fully restores normal update behavior. Nothing is deleted or uninstalled.

**Will this work on a company or school laptop?**
Domain-joined machines managed by Group Policy may overwrite these same registry keys on their next policy refresh, so `--lock` is intended for personal machines you administer yourself.

**Does BandwidthGuard collect any data or phone home?**
No. It has zero network calls and zero external dependencies — every line of what it does is in the single `main.go` file in this repository.

**Why does it need administrator rights?**
Every change it makes — stopping services, editing `HKLM` registry keys, disabling scheduled tasks — requires elevation on Windows. There's no way around this for a tool that does what BandwidthGuard does.

**How is this different from just running `sc stop wuauserv` manually?**
Stopping the service alone doesn't stick — Windows' own scheduled tasks and the Update Medic service will restart it. BandwidthGuard disables the service, the tasks that revive it, and the policy that governs it, all in one step, and can undo all three just as cleanly.

**Is this safe to run?**
Yes — it only touches the specific services, tasks, and registry values listed above, all of which are standard, documented Windows components. The full source is in this repository for anyone to review before running it.

## Author

**BandwidthGuard** is developed and maintained by **Kanade Shiraishi**.

- GitHub: [@kanade-shiraishi](https://github.com/kanade-shiraishi)

If BandwidthGuard was useful to you, a star on this repository helps other people find it.

## License

MIT © Yoshie Shiraishi — see [LICENSE](LICENSE) for details.

#<meta name="msvalidate.01" content="03176C7F130CA75C6703F783A39E38F1" />
