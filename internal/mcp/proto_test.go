package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// protoSession manages a running vidscribe --mcp subprocess.
type protoSession struct {
	cmd    *exec.Cmd
	enc    *json.Encoder
	dec    *bufio.Scanner
	nextID int
}

func startMCPServer(t *testing.T) *protoSession {
	t.Helper()

	// Build the binary into a temp file so we test the real binary, not go run.
	bin := filepath.Join(t.TempDir(), "vidscribe-proto-test")
	if out, err := exec.Command("go", "build", "-o", bin, "../..").CombinedOutput(); err != nil {
		t.Skipf("could not build binary: %v\n%s", err, out)
	}

	cmd := exec.Command(bin, "--mcp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start mcp server: %v", err)
	}

	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	s := &protoSession{
		cmd: cmd,
		enc: json.NewEncoder(stdin),
		dec: bufio.NewScanner(stdout),
	}
	s.dec.Buffer(make([]byte, 1<<20), 1<<20)
	return s
}

type rpcMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  any             `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *protoSession) send(method string, params any) {
	s.nextID++
	_ = s.enc.Encode(rpcMsg{
		JSONRPC: "2.0",
		ID:      &s.nextID,
		Method:  method,
		Params:  params,
	})
}

func (s *protoSession) notify(method string, params any) {
	_ = s.enc.Encode(rpcMsg{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

func (s *protoSession) recv(t *testing.T) rpcMsg {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if s.dec.Scan() {
			var msg rpcMsg
			if err := json.Unmarshal(s.dec.Bytes(), &msg); err != nil {
				t.Fatalf("unmarshal response: %v\nraw: %s", err, s.dec.Bytes())
			}
			return msg
		}
	}
	t.Fatal("timeout waiting for MCP response")
	return rpcMsg{}
}

// handshake performs the MCP initialization sequence.
func (s *protoSession) handshake(t *testing.T) {
	t.Helper()
	s.send("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "vidscribe-test", "version": "1.0"},
	})
	resp := s.recv(t)
	if resp.Error != nil {
		t.Fatalf("initialize error: %v", resp.Error.Message)
	}
	s.notify("notifications/initialized", map[string]any{})
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestProto_ToolsList(t *testing.T) {
	s := startMCPServer(t)
	s.handshake(t)

	s.send("tools/list", map[string]any{})
	resp := s.recv(t)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %s", resp.Error.Message)
	}

	var result struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema struct {
				Properties map[string]any `json:"properties"`
				Required   []string       `json:"required"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("parse tools/list result: %v", err)
	}

	// Verify exactly the 3 expected tools are registered.
	wantTools := map[string]bool{
		"transcribe_video":     false,
		"check_dependencies":   false,
		"list_supported_sites": false,
	}
	for _, tool := range result.Tools {
		wantTools[tool.Name] = true
	}
	for name, found := range wantTools {
		if !found {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
	if t.Failed() {
		t.Logf("registered tools: %v", func() []string {
			var names []string
			for _, tool := range result.Tools {
				names = append(names, tool.Name)
			}
			return names
		}())
	}

	// Verify transcribe_video schema: url is required, others are optional.
	for _, tool := range result.Tools {
		if tool.Name != "transcribe_video" {
			continue
		}
		// url must be required
		hasURLRequired := false
		for _, r := range tool.InputSchema.Required {
			if r == "url" {
				hasURLRequired = true
			}
		}
		if !hasURLRequired {
			t.Errorf("transcribe_video: 'url' not in required list %v", tool.InputSchema.Required)
		}
		// optional params must exist in properties
		for _, param := range []string{"model", "language", "cookies_browser", "cookies_file", "engine", "format"} {
			if _, ok := tool.InputSchema.Properties[param]; !ok {
				t.Errorf("transcribe_video: optional param %q missing from schema", param)
			}
		}
	}
}

func TestProto_CheckDependencies(t *testing.T) {
	s := startMCPServer(t)
	s.handshake(t)

	s.send("tools/call", map[string]any{
		"name":      "check_dependencies",
		"arguments": map[string]any{},
	})
	resp := s.recv(t)
	if resp.Error != nil {
		t.Fatalf("tools/call error: %s", resp.Error.Message)
	}

	text := extractText(t, resp.Result)
	for _, want := range []string{"uvx", "yt-dlp"} {
		if !strings.Contains(text, want) {
			t.Errorf("check_dependencies: missing %q in:\n%s", want, text)
		}
	}
	t.Logf("check_dependencies response:\n%s", text)
}

func TestProto_ListSupportedSites(t *testing.T) {
	if !uvxAvailable() {
		t.Skip("uvx not in PATH")
	}
	s := startMCPServer(t)
	s.handshake(t)

	s.send("tools/call", map[string]any{
		"name":      "list_supported_sites",
		"arguments": map[string]any{},
	})
	resp := s.recv(t)
	if resp.Error != nil {
		t.Fatalf("tools/call error: %s", resp.Error.Message)
	}

	text := extractText(t, resp.Result)
	if !strings.Contains(strings.ToLower(text), "youtube") {
		t.Errorf("list_supported_sites: 'youtube' not found in output")
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func extractText(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("parse call result: %v\nraw: %s", err, raw)
	}
	var sb strings.Builder
	for _, c := range result.Content {
		if c.Type == "text" {
			fmt.Fprint(&sb, c.Text)
		}
	}
	return sb.String()
}
