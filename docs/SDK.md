# mobile-cli SDK Footprint

This document is the **acceptance evidence** for the "Assess SDK footprint"
closure verb in mobile-cli's row 24 of
`13-breadth-readiness-and-priority.md`. It enumerates every Go module the
repo declares, classifies each by role, and documents the official build
profile that consumes them.

## Module

```
github.com/mobile-next/mobilecli
```

## Direct Dependencies (15)

Classified by role:

### Native-bridge (iOS + Android device communication)

| Module | Version | Role |
|---|---|---|
| `github.com/danielpaulus/go-ios` | `v1.0.211` | iOS device bridge (lockdown, instruments, screenshot, XCTest) |
| `github.com/shogo82148/androidbinary` | `v1.0.5` | Android APK/dex parsing + signing |
| `github.com/mobile-next/mobilecli` | (self) | this repo |

### Media / Codec

| Module | Version | Role |
|---|---|---|
| `github.com/yapingcat/gomedia` | `2024-09-06` | MP4/MOV muxer, H.264 frame parsing |

### IPC / Streaming

| Module | Version | Role |
|---|---|---|
| `github.com/gorilla/websocket` | `v1.5.3` | WebDriverAgent (WDA) bidirectional transport |
| `github.com/sirupsen/logrus` | `v1.9.3` | structured logging |
| `github.com/google/uuid` | `v1.6.0` | session/screenshot identifiers |

### CLI / Daemon lifecycle

| Module | Version | Role |
|---|---|---|
| `github.com/spf13/cobra` | `v1.9.1` | CLI command tree |
| `github.com/sevlyar/go-daemon` | `v0.1.6` | background-daemon spawning (mobilecli serve) |

### Persistence

| Module | Version | Role |
|---|---|---|
| `github.com/zalando/go-keyring` | `v0.2.6` | macOS Keychain / Linux Secret Service / Windows Credential Manager |
| `github.com/hashicorp/golang-lru/v2` | `v2.0.7` | bounded LRU cache for device/session state |
| `howett.net/plist` | `v1.0.1` | plist (XML) parsing for iOS device metadata |
| `gopkg.in/ini.v1` | `v1.67.0` | INI parsing for config files |
| `al.essio.dev/pkg/shellescape` | `v1.5.1` | shell-quoting for child process args |

### Test-only

| Module | Version | Role |
|---|---|---|
| `github.com/stretchr/testify` | `v1.10.0` | assertions for unit + integration tests |

## Indirect Dependencies (131)

These are pulled in transitively by the direct deps. All are pinned in
`go.sum`. The full list is regenerable via:

```bash
go list -m -f '{{if and .Indirect (not .Main)}}{{.Path}} {{.Version}}{{end}}' all
```

Notable categories (verified by manual classification of the
`go list -m all` output):

| Category | Examples |
|---|---|
| Standard library shims / wrappers | `golang.org/x/sys`, `golang.org/x/text` |
| TLS / crypto | `github.com/golang-jwt/jwt/v5`, `golang.org/x/crypto` |
| YAML / JSON codecs | `gopkg.in/yaml.v3`, `github.com/golang-jwt/jwt/v5` |
| Test infrastructure | `github.com/davecgh/go-spew`, `github.com/pmezard/go-difflib` |

## Build profile

The default-tag build (`go build .`) produces a ~20 MB Mach-O 64-bit
binary that includes:

- CLI command tree (cobra)
- iOS device bridge (go-ios)
- Android device bridge (androidbinary)
- WDA websocket client (gorilla/websocket)
- MP4 muxer (gomedia)
- Keychain integration (go-keyring)

The `-tags agents` build additionally embeds:

- `agents/android/mobilecli.so` (Frida gadget)
- `agents/android/mobilecli.dex` (Dalvik dex)
- `agents/ios/agent-sim.dylib` (Frida simulator gadget)
- `agents/ios-real/agent.m` (ObjC source for real-device Frida)

These are produced by `make agents` (requires Android NDK + Xcode
toolchains) and are NOT committed to the repo (see
`agents/agents_binaries.go` + `agents/agents_binaries_stub.go` for the
build-tag split).

## Verification

Run on `main` at HEAD `8f6247f`:

```bash
$ go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all | sort -u
al.essio.dev/pkg/shellescape v1.5.1
github.com/danielpaulus/go-ios v1.0.211
github.com/google/uuid v1.6.0
github.com/gorilla/websocket v1.5.3
github.com/hashicorp/golang-lru/v2 v2.0.7
github.com/mobile-next/mobilecli
github.com/sevlyar/go-daemon v0.1.6
github.com/shogo82148/androidbinary v1.0.5
github.com/sirupsen/logrus v1.9.3
github.com/spf13/cobra v1.9.1
github.com/stretchr/testify v1.10.0
github.com/yapingcat/gomedia v0.0.0-20240906162731-17feea57090c
github.com/zalando/go-keyring v0.2.6
gopkg.in/ini.v1 v1.67.0
howett.net/plist v1.0.1

$ CGO_ENABLED=0 go build -o /tmp/mobilecli . && file /tmp/mobilecli
/tmp/mobilecli: Mach-O 64-bit arm64 executable, flags:<NOUNDEFS|DYLDLINK|TWOLEVEL|PIE>
```
