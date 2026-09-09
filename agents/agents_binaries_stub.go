package agents

import _ "embed"

// Default (no build tag) placeholders. The real binaries are produced by
// `make -C agents/android` (Java → dex, C → .so) and `make -C agents/ios`
// (Objective-C → .dylib) and are not committed to the repository.
//
// To produce a deployable CLI that can push the webview agent to a device,
// build with:
//
//	make agents
//	go build -tags agents -ldflags="-s -w"
//
// When built without the `agents` tag, these variables are nil and any code
// path that consumes them (devices/android_webview.go,
// devices/ios_webview.go) will receive empty bytes. Callers already validate
// the data path; a nil embed causes a real error at the call site rather than
// a silent build failure.
//
// android/mobilecli.so
var AndroidMobilecliSO []byte

// android/mobilecli.dex
var AndroidMobilecliDEX []byte

// ios/agent-sim.dylib
var IOSAgentSimDylib []byte
