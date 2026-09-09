//go:build agents

package agents

import _ "embed"

// android/mobilecli.so and android/mobilecli.dex are produced by
// `make -C agents/android` from the Java sources in agents/android/java and the
// JNI source agents/android/jvmti_agent.c. ios/agent-sim.dylib is produced by
// `make -C agents/ios`.
//
// These binaries are only embedded when the `agents` build tag is set, so the
// default `go test` / `go build` path does not require the toolchains. See
// agents_binaries_stub.go for the no-tag default.
//
//go:embed android/mobilecli.so
var AndroidMobilecliSO []byte

//go:embed android/mobilecli.dex
var AndroidMobilecliDEX []byte

//go:embed ios/agent-sim.dylib
var IOSAgentSimDylib []byte
