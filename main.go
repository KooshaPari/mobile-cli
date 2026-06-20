package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mobile-next/mobilecli/cli"
	"github.com/mobile-next/mobilecli/commands"
	"github.com/mobile-next/mobilecli/daemon"
	"github.com/mobile-next/mobilecli/devices"
	"github.com/mobile-next/mobilecli/utils"
)

// eidolonCommandsAdapter bridges the package-main EidolonDispatcher (which
// returns main.EidolonResult) to the commands.EidolonDispatcher interface
// (which uses commands.EidolonResultEnvelope, since Go does not allow
// subpackages to import package main). The mapping is trivial: both
// envelopes have identical JSON shape so callers see the same Data on
// success and the same Error string on failure.
type eidolonCommandsAdapter struct{ inner EidolonDispatcher }

func (a eidolonCommandsAdapter) Name() string { return a.inner.Name() }
func (a eidolonCommandsAdapter) Dispatch(ctx context.Context, method string, params map[string]any) commands.EidolonResultEnvelope {
	res := a.inner.Dispatch(ctx, method, params)
	return commands.EidolonResultEnvelope{OK: res.OK, Data: res.Data, Error: res.Error}
}

// parseEidolonEndpoint scans os.Args for the --eidolon-endpoint flag (either
// `--eidolon-endpoint=VALUE` or `--eidolon-endpoint VALUE`) and returns the
// value or "" if not present. We parse it here because cobra lives in the cli
// subpackage and Go does not allow subpackages to import package main, so the
// dispatcher construction (which lives in main) cannot be wired from cli/.
// The flag is also registered with cobra in cli/root.go so --help shows it.
func parseEidolonEndpoint(args []string) string {
	const flag = "--eidolon-endpoint"
	for i, a := range args {
		if a == flag {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(a, flag+"=") {
			return strings.TrimPrefix(a, flag+"=")
		}
	}
	return ""
}

func main() {
	// wire Eidolon dispatcher (if requested) before cobra parses flags,
	// so device-targeting commands see the configured dispatcher on first use.
	if endpoint := parseEidolonEndpoint(os.Args[1:]); endpoint != "" {
		d := NewEidolonDispatcherFromEndpoint(endpoint)
		commands.SetEidolonDispatcher(eidolonCommandsAdapter{inner: d})
		utils.Verbose("eidolon: dispatcher configured (%s)", d.Name())
	}

	// daemon child sets up its own signal handling in server.StartServer
	if daemon.IsChild() {
		if err := cli.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// create shutdown hook for cleanup tracking
	hook := devices.NewShutdownHook()
	commands.SetShutdownHook(hook)

	// setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// run command in goroutine
	done := make(chan error, 1)
	go func() {
		done <- cli.Execute()
	}()

	// wait for command completion or signal
	select {
	case <-sigChan:
		// cleanup resources on signal
		hook.Shutdown()
		os.Exit(0)
	case err := <-done:
		// cleanup resources on normal exit
		hook.Shutdown()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
