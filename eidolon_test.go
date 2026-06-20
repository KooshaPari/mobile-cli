package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// mockEidolonTransport is an in-memory EidolonTransport used by tests.
type mockEidolonTransport struct {
	name      string
	method    string
	params    map[string]any
	result    EidolonResult
	callErr   error
	callCount int
}

func (m *mockEidolonTransport) Name() string { return m.name }

func (m *mockEidolonTransport) Call(_ context.Context, method string, params map[string]any) (EidolonResult, error) {
	m.callCount++
	m.method = method
	m.params = params
	if m.callErr != nil {
		return EidolonResult{}, m.callErr
	}
	return m.result, nil
}

func TestNullEidolonDispatcherReturnsError(t *testing.T) {
	d := NewNullEidolonDispatcher()

	if d.Name() != "null" {
		t.Fatalf("expected name=%q, got %q", "null", d.Name())
	}

	res := d.Dispatch(context.Background(), "io.tap", map[string]any{"x": 1, "y": 2})

	if res.OK {
		t.Fatalf("NullEidolonDispatcher must return ok=false, got ok=true (data=%v)", res.Data)
	}
	if res.Error == "" {
		t.Fatalf("NullEidolonDispatcher must return a non-empty error message")
	}

	// Sanity: the error mentions reachability so users know what happened.
	if !contains(res.Error, "eidolon not reachable") {
		t.Fatalf("expected error to mention 'eidolon not reachable', got %q", res.Error)
	}
}

func TestMcpEidolonDispatcherForwardsViaMockTransport(t *testing.T) {
	mock := &mockEidolonTransport{
		name: "mock",
		result: EidolonResult{
			OK:   true,
			Data: map[string]any{"status": "tapped", "x": 100, "y": 200},
		},
	}
	d := NewMcpEidolonDispatcher(mock)

	params := map[string]any{"deviceId": "iphone-15", "x": 100, "y": 200}
	res := d.Dispatch(context.Background(), "io.tap", params)

	if !res.OK {
		t.Fatalf("expected ok=true, got ok=false (error=%q)", res.Error)
	}
	if mock.callCount != 1 {
		t.Fatalf("expected transport to be called once, got %d calls", mock.callCount)
	}
	if mock.method != "io.tap" {
		t.Fatalf("expected forwarded method=io.tap, got %q", mock.method)
	}
	if mock.params["deviceId"] != "iphone-15" || mock.params["x"] != 100 || mock.params["y"] != 200 {
		t.Fatalf("forwarded params do not match input: %v", mock.params)
	}

	data, ok := res.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected Data to be map[string]any, got %T", res.Data)
	}
	if data["status"] != "tapped" {
		t.Fatalf("expected forwarded data.status=tapped, got %v", data["status"])
	}
}

func TestMcpEidolonDispatcherPropagatesTransportError(t *testing.T) {
	mock := &mockEidolonTransport{
		name:    "mock-err",
		callErr: errors.New("connection refused"),
	}
	d := NewMcpEidolonDispatcher(mock)

	res := d.Dispatch(context.Background(), "io.tap", map[string]any{"x": 0, "y": 0})

	if res.OK {
		t.Fatalf("expected ok=false when transport returns an error")
	}
	if res.Error != "connection refused" {
		t.Fatalf("expected transport error to be propagated, got %q", res.Error)
	}
}

func TestEidolonDispatcherInterfaceCompat(t *testing.T) {
	// Compile-time check: both concrete dispatchers implement EidolonDispatcher.
	var _ EidolonDispatcher = (*NullEidolonDispatcher)(nil)
	var _ EidolonDispatcher = (*McpEidolonDispatcher)(nil)

	// Runtime check: NewEidolonDispatcherFromEndpoint returns the right concrete type.
	nullDispatcher := NewEidolonDispatcherFromEndpoint("")
	if _, ok := nullDispatcher.(*NullEidolonDispatcher); !ok {
		t.Fatalf("empty endpoint should return *NullEidolonDispatcher, got %T", nullDispatcher)
	}

	stdioDispatcher := NewEidolonDispatcherFromEndpoint("/usr/local/bin/eidolon-mcp")
	if mcp, ok := stdioDispatcher.(*McpEidolonDispatcher); !ok {
		t.Fatalf("path endpoint should return *McpEidolonDispatcher, got %T", stdioDispatcher)
	} else {
		if _, ok := mcp.transport.(*StdioEidolonTransport); !ok {
			t.Fatalf("path endpoint should produce StdioEidolonTransport, got %T", mcp.transport)
		}
	}

	httpDispatcher := NewEidolonDispatcherFromEndpoint("http://localhost:3100")
	if mcp, ok := httpDispatcher.(*McpEidolonDispatcher); !ok {
		t.Fatalf("http endpoint should return *McpEidolonDispatcher, got %T", httpDispatcher)
	} else {
		if _, ok := mcp.transport.(*HttpEidolonTransport); !ok {
			t.Fatalf("http endpoint should produce HttpEidolonTransport, got %T", mcp.transport)
		}
	}

	// The result envelope round-trips through JSON (mirrors the wire format the
	// MCP server speaks).
	original := EidolonResult{OK: true, Data: map[string]any{"foo": "bar"}}
	buf, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded EidolonResult
	if err := json.Unmarshal(buf, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !decoded.OK || decoded.Error != "" {
		t.Fatalf("expected ok=true after round-trip, got %+v", decoded)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
