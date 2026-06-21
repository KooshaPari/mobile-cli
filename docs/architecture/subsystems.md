# Subsystems — mobile-cli

ADR-038 cross-link: see [ADR-038: Hexagonal port-adapter L4 policy](https://github.com/KooshaPari/phenotype-apps/blob/main/docs/adr/2026-06-18/ADR-038-hexagonal-port-adapter-l4-policy.md) for the canonical input/output port contract.

> L7 subsystem decomposition. Bounded contexts, ports, owned data, external
> dependencies, and failure modes for the mobile-cli Go monorepo (device
> automation CLI for iOS / Android / Simulator). Companion to `README.md`.
> Initial decomposition 2026-06-21 (v16 cycle-6 T1).

## Subsystem map

| Subsystem | Path | Responsibility | Owned data | Critical? |
|---|---|---|---|---|
| CLI entry / commands | `cli/`, `commands/`, `root.go` | Parse argv, route sub-commands, render output | argv config, exit code | yes |
| Daemon (gRPC server) | `daemon/daemon.go`, `server/` | Long-running gRPC server for remote MCP clients | device cache, session table | yes |
| Devices (iOS / Android) | `devices/ios*.go`, `devices/android*.go`, `devices/simulator.go` | Per-platform device automation adapters | per-device session, screenshot cache | yes |
| Agents (iOS / Android worker pools) | `agents/`, `eidolon.go` | Worker pool for screenshot, accessibility, key injection | worker state, gesture queue | no |
| Eidolon bridge | `eidolon.go`, `eidolon_test.go` | `EidolonTransport` adapter that delegates Eidolon method calls to this CLI | endpoint URL, EidolonMethod registry | no |

## Port catalogue

### Input ports (consumed)

- `mobile-mcp::EidolonTransport` — JSON-RPC over TCP from `mobile-mcp` shim.
- `pheno-config::Config` (via `Configra`) — layered config.
- `pheno-errors::Error` envelope.
- `pheno-tracing` OTLP exporter.
- `adb` (Android SDK) — used by `devices/android*.go`.
- `xcrun simctl` / `idevice*` (iOS toolchain) — used by `devices/ios*.go`.

### Output ports (produced)

- CLI sub-commands (`devices list`, `screenshot`, `boot`, etc.) — public API.
- gRPC server on `localhost:port` — consumed by `mobile-mcp`.
- Eidolon method dispatch — backwards-compat with Eidolon stage.
- Telemetry events on every command (via `pheno-tracing`).

## External dependencies

| Dependency | Kind | Used by |
|---|---|---|
| `mobile-mcp` | Go module (subprocess OR TCP) | Eidolon shim |
| `eidolon` | Go module (path-dep into `../Eidolon`) | Eidolon bridge |
| `pheno-config` | Go module (path-dep into `pheno-config`) | config cascade |
| `pheno-errors` | Go module | error envelope |
| `pheno-tracing` | Go module (path-dep into `pheno-tracing`) | OTLP spans |
| `adb` | Android SDK | android adapter |
| `xcrun simctl`, `idevice_id`, `ios-deploy` | iOS toolchain | iOS adapter |
| `grpc-go` | Go module | daemon transport |

## Failure modes

| Subsystem | Failure | Detection | Recovery |
|---|---|---|---|
| CLI entry | sub-command not found | argv parse fail | print usage; exit 2 |
| Daemon | gRPC port already in use | bind EADDRINUSE | retry with next free port; max 3 |
| Daemon | client timeout | context deadline | cancel; release session |
| Devices (iOS) | device disconnected | `idevice_id` returns empty | refresh device list; emit `DeviceGone` |
| Devices (iOS) | WebDriverAgent crash | WDA process exit | restart WDA bundle; re-attach |
| Devices (Android) | adb offline | `adb get-state` → `offline` | reconnect; refresh device list |
| Devices (Android) | UiAutomator dump timeout | `uiautomator dump` > 30s | retry; surface as `CommandError` |
| Devices (simulator) | runtime missing | `xcrun simctl list` empty | emit `EnvironmentError` |
| Agents | worker pool exhausted | all workers busy > 60s | backpressure; return partial result |
| Eidolon bridge | endpoint unreachable | TCP RST or 5xx | exponential backoff; max 3 retries |
| Eidolon bridge | schema drift | EidolonMethod not in registry | log warn; skip dispatch; continue |

## Change log

- 2026-06-21 — initial decomposition (v16 cycle-6 T1, L7). ADR-038 cross-link added.
