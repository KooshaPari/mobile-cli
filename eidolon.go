package main

// EidolonStage integration for mobilecli.
//
// When the user passes --eidolon-endpoint=<path|url>, every device-targeting
// command first tries to dispatch the operation through an Eidolon MCP server.
// If the dispatch succeeds the response is returned as-is; on any failure the
// command transparently falls back to the native iOS/Android implementation.
//
// This mirrors the adapter in
// /Users/kooshapari/CodeProjects/Phenotype/repos/agent-platform/ports/adapters/eidolon.ts
// (EidolonStage + McpStdioTransport + McpHttpTransport + NullTransport), which
// is the canonical T16.4 substrate for device automation across mobile, desktop
// and sandbox modalities.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

// EidolonResult is the canonical result envelope returned by every Eidolon MCP
// operation, regardless of transport.
type EidolonResult struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// EidolonTransport is the transport-agnostic interface used to talk to an
// Eidolon MCP server. Implementations include stdio (spawn a child process and
// speak JSON-RPC over stdin/stdout) and HTTP (POST /call on a remote server).
type EidolonTransport interface {
	Name() string
	Call(ctx context.Context, method string, params map[string]any) (EidolonResult, error)
}

// EidolonDispatcher is the public interface commands use to forward
// operations. It is transport-agnostic and safe to call concurrently.
type EidolonDispatcher interface {
	Name() string
	Dispatch(ctx context.Context, method string, params map[string]any) EidolonResult
}

// NullEidolonDispatcher is the default no-op dispatcher. It returns ok=false
// for every call so callers transparently fall back to the native iOS/Android
// implementation. This mirrors the TS NullTransport in eidolon.ts.
type NullEidolonDispatcher struct{}

// NewNullEidolonDispatcher returns a NullEidolonDispatcher.
func NewNullEidolonDispatcher() *NullEidolonDispatcher {
	return &NullEidolonDispatcher{}
}

func (NullEidolonDispatcher) Name() string { return "null" }

func (NullEidolonDispatcher) Dispatch(_ context.Context, _ string, _ map[string]any) EidolonResult {
	return EidolonResult{OK: false, Error: "eidolon not reachable: no endpoint configured"}
}

// McpEidolonDispatcher forwards calls through an EidolonTransport. It is the
// concrete dispatcher used when --eidolon-endpoint is set.
type McpEidolonDispatcher struct {
	transport EidolonTransport
}

// NewMcpEidolonDispatcher wraps a transport in a dispatcher.
func NewMcpEidolonDispatcher(transport EidolonTransport) *McpEidolonDispatcher {
	return &McpEidolonDispatcher{transport: transport}
}

func (d *McpEidolonDispatcher) Name() string { return d.transport.Name() }

func (d *McpEidolonDispatcher) Dispatch(ctx context.Context, method string, params map[string]any) EidolonResult {
	result, err := d.transport.Call(ctx, method, params)
	if err != nil {
		return EidolonResult{OK: false, Error: err.Error()}
	}
	return result
}

// StdioEidolonTransport spawns an Eidolon MCP server child process and speaks
// JSON-RPC 2.0 over its stdin/stdout. Each Call spawns a fresh process and
// terminates it after a single request/response cycle; this is sufficient for
// the flag-driven mode and avoids having to manage persistent sessions.
type StdioEidolonTransport struct {
	binaryPath string
}

// NewStdioEidolonTransport returns a transport that spawns the given binary
// for each call.
func NewStdioEidolonTransport(binaryPath string) *StdioEidolonTransport {
	return &StdioEidolonTransport{binaryPath: binaryPath}
}

func (t *StdioEidolonTransport) Name() string { return "stdio:" + t.binaryPath }

func (t *StdioEidolonTransport) Call(ctx context.Context, method string, params map[string]any) (EidolonResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(callCtx, t.binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return EidolonResult{}, fmt.Errorf("open stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return EidolonResult{}, fmt.Errorf("open stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return EidolonResult{}, fmt.Errorf("spawn %s: %w", t.binaryPath, err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	if err := json.NewEncoder(stdin).Encode(req); err != nil {
		return EidolonResult{}, fmt.Errorf("encode request: %w", err)
	}
	_ = stdin.Close()

	var raw json.RawMessage
	if err := json.NewDecoder(stdout).Decode(&raw); err != nil {
		return EidolonResult{}, fmt.Errorf("decode response: %w", err)
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return EidolonResult{}, fmt.Errorf("parse jsonrpc response: %w", err)
	}
	if rpcResp.Error != nil {
		return EidolonResult{OK: false, Error: rpcResp.Error.Message}, nil
	}
	if len(rpcResp.Result) == 0 {
		return EidolonResult{OK: true, Data: nil}, nil
	}
	var data any
	if err := json.Unmarshal(rpcResp.Result, &data); err != nil {
		return EidolonResult{}, fmt.Errorf("decode result payload: %w", err)
	}
	return EidolonResult{OK: true, Data: data}, nil
}

// HttpEidolonTransport POSTs JSON-RPC requests to a running Eidolon MCP server
// over HTTP. The server is expected to expose POST {baseUrl}/call accepting
// {"method":..., "params":...} and returning {"ok":..., "data":..., "error":...}.
type HttpEidolonTransport struct {
	baseURL string
	client  *http.Client
}

// NewHttpEidolonTransport returns an HTTP transport pointing at baseURL.
func NewHttpEidolonTransport(baseURL string) *HttpEidolonTransport {
	return &HttpEidolonTransport{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *HttpEidolonTransport) Name() string { return "http:" + t.baseURL }

func (t *HttpEidolonTransport) Call(ctx context.Context, method string, params map[string]any) (EidolonResult, error) {
	body, err := json.Marshal(map[string]any{"method": method, "params": params})
	if err != nil {
		return EidolonResult{}, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/call", bytes.NewReader(body))
	if err != nil {
		return EidolonResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return EidolonResult{}, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return EidolonResult{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return EidolonResult{OK: false, Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(raw))}, nil
	}

	var result EidolonResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return EidolonResult{}, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

// NewEidolonDispatcherFromEndpoint constructs the right dispatcher for an
// --eidolon-endpoint value. Paths containing "://" or starting with "http" are
// treated as HTTP URLs; everything else is treated as a stdio binary path.
// An empty endpoint returns a NullEidolonDispatcher.
func NewEidolonDispatcherFromEndpoint(endpoint string) EidolonDispatcher {
	if endpoint == "" {
		return NewNullEidolonDispatcher()
	}
	if len(endpoint) >= 4 && endpoint[:4] == "http" {
		return NewMcpEidolonDispatcher(NewHttpEidolonTransport(endpoint))
	}
	return NewMcpEidolonDispatcher(NewStdioEidolonTransport(endpoint))
}
