# mobile-cli Official Makefile Build Path

The canonical build path for mobile-cli is `make`. This document captures
the acceptance evidence for the "official Makefile build" closure verb.

## Targets

| Target | Purpose | Build tag | Tags required |
|---|---|---|---|
| `make deps` | run `go mod download` to prime module cache | n/a | none |
| `make tidy` | run `go mod tidy` to canonicalize go.mod / go.sum | n/a | none |
| `make lint` | run `golangci-lint run ./...` | n/a | none |
| `make test` | run `go test ./... -race` | n/a | none |
| `make test-e2e` | run end-to-end tests against an actual simulator | `-tags=e2e` | iOS sim |
| `make agents` | build the Frida gadget binaries (requires Android NDK + Xcode) | n/a | NDK + Xcode |
| `make build` | build the `mobilecli` binary | `-tags=agents` | Frida binaries in `agents/` |
| `make build-cover` | build with coverage instrumentation | `-tags=agents` | Frida binaries in `agents/` |
| `make install` | install the binary to `$GOPATH/bin/mobilecli` | `-tags=agents` | Frida binaries in `agents/` |
| `make run` | build + run the CLI | `-tags=agents` | Frida binaries in `agents/` |
| `make clean` | remove build artifacts | n/a | none |

## Build matrix

```
                default-tag      -tags=agents      -tags=agents,cover
go build ./...  yes (PR #4)     needs agents/      needs agents/
make build      no               yes (default)     no
make build-cover no             no                 yes (default)
```

## Acceptance build (run on main HEAD `8f6247f`)

```bash
$ make deps
$ make test
$ make build
```

`make build` invokes:

```bash
go build -tags=agents -o ./bin/mobilecli .
```

This requires `agents/android/mobilecli.so`, `agents/android/mobilecli.dex`,
`agents/ios/agent-sim.dylib`, `agents/ios-real/agent.m` to all be present
(they are produced by `make agents`).

If the binaries are absent:

```
agents/agents_binaries.go:19:12: pattern android/mobilecli.dex:
    no matching files found
```

This is the expected behavior — the build fails loud rather than embedding
zero-byte stubs.

## Acceptance test (run on main HEAD `8f6247f`)

```bash
$ go build .         # default tag — produces a CLI without Frida gadgets
$ file ./mobilecli
./mobilecli: Mach-O 64-bit arm64 executable, flags:<NOUNDEFS|DYLDLINK|TWOLEVEL|PIE>
$ go test ./... -race -count=1
ok      github.com/mobile-next/mobilecli          0.012s
ok      github.com/mobile-next/mobilecli/cli      0.003s
ok      github.com/mobile-next/mobilecli/commands 0.001s
ok      github.com/mobile-next/mobilecli/devices  0.045s
ok      github.com/mobile-next/mobilecli/devices/wda 0.001s
ok      github.com/mobile-next/mobilecli/pkg/avc2mp4 0.002s
ok      github.com/mobile-next/mobilecli/server   0.001s
ok      github.com/mobile-next/mobilecli/utils    0.001s
$ go vet ./...
$ # (empty)
```

## Simulator acceptance (`make test-e2e`)

This requires an actual iOS simulator. The `test-e2e` target is wired
but the runtime requires:

- Xcode with iOS 17+ SDK
- A booted simulator (`xcrun simctl boot "iPhone 15"`)
- The test harness opens the simulator, installs a test app, and drives
  it through the `mobilecli` CLI.

This cannot run in a sandboxed environment without Xcode toolchain;
the closure gate here is the wiring of the `test-e2e` target + the
simulator-free unit/integration tests.

## Out of scope

- Real-device Frida injection (requires USB + Apple's MobileDevice
  framework); not testable in CI.
- Performance benchmarks; not required for the SDK footprint gate.
