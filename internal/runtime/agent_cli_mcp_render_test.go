package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readAgentCLIMCPDoc(t *testing.T, home, rel string) any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, rel))
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func normalizeJSON(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestWriteAgentCLIMCPConfigOpenCodeServers(t *testing.T) {
	home := t.TempDir()
	entries := map[string]any{
		"http":      map[string]any{"type": "http", "url": "http://127.0.0.1:1/mcp", "headers": map[string]any{"Authorization": "Bearer x"}},
		"http-bare": map[string]any{"url": "http://127.0.0.1:2/mcp"},
		"stdio-str": map[string]any{"command": "node", "args": []string{"a.js", "--flag"}, "env": map[string]any{"K": "V"}, "cwd": "/w"},
		"stdio-any": map[string]any{"command": "py", "args": []any{"b.py"}},
		"empty":     map[string]any{},
		"invalid":   "not-a-map",
	}
	if err := WriteAgentCLIMCPConfig(AgentCLIOpenCode, HostAgentCLIHome(home), entries, AgentCLIMCPOptions{}); err != nil {
		t.Fatal(err)
	}
	got := readAgentCLIMCPDoc(t, home, ".config/opencode/opencode.json").(map[string]any)["mcp"]
	want := normalizeJSON(t, map[string]any{
		"http":      map[string]any{"type": "remote", "url": "http://127.0.0.1:1/mcp", "enabled": true, "timeout": 600000, "headers": map[string]any{"Authorization": "Bearer x"}},
		"http-bare": map[string]any{"type": "remote", "url": "http://127.0.0.1:2/mcp", "enabled": true, "timeout": 600000},
		"stdio-str": map[string]any{"type": "local", "command": []any{"node", "a.js", "--flag"}, "enabled": true, "timeout": 600000, "environment": map[string]any{"K": "V"}},
		"stdio-any": map[string]any{"type": "local", "command": []any{"py", "b.py"}, "enabled": true, "timeout": 600000},
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("opencode mcp:\n got %v\nwant %v", got, want)
	}
}

func TestWriteAgentCLIMCPConfigDeepSeekInserts(t *testing.T) {
	home := t.TempDir()
	longID := "agentflow.platform/" + strings.Repeat("x", 40)
	entries := map[string]any{
		"a-http":  map[string]any{"url": "http://127.0.0.1:1/mcp", "headers": map[string]any{"H": "1"}},
		"b.stdio": map[string]any{"command": "node", "args": []string{"s.js"}, "env": map[string]any{"K": "V"}, "cwd": "/w"},
		longID:    map[string]any{"command": "bin"},
		"skip":    map[string]any{},
	}
	if err := WriteAgentCLIMCPConfig(AgentCLIDeepSeek, HostAgentCLIHome(home), entries, AgentCLIMCPOptions{}); err != nil {
		t.Fatal(err)
	}
	longName := ("agentflow_platform_" + strings.Repeat("x", 40))[:32]
	want := normalizeJSON(t, []any{map[string]any{"insert": []any{
		map[string]any{"id": "agentflow-mcp-a-http", "name": "@deepseek-ai/dsh-mcp-client", "config": map[string]any{
			"serverName": "a-http", "toolCallTimeoutMs": 600000, "transport": "streamable-http", "url": "http://127.0.0.1:1/mcp", "headers": map[string]any{"H": "1"},
		}},
		map[string]any{"id": "agentflow-mcp-" + longName, "name": "@deepseek-ai/dsh-mcp-client", "config": map[string]any{
			"serverName": longName, "toolCallTimeoutMs": 600000, "transport": "stdio", "command": "bin",
		}},
		map[string]any{"id": "agentflow-mcp-b_stdio", "name": "@deepseek-ai/dsh-mcp-client", "config": map[string]any{
			"serverName": "b_stdio", "toolCallTimeoutMs": 600000, "transport": "stdio", "command": "node", "args": []any{"s.js"}, "env": map[string]any{"K": "V"}, "cwd": "/w",
		}},
	}}})
	rel := agentCLIMCPPaths[AgentCLIDeepSeek]
	if got := readAgentCLIMCPDoc(t, home, rel); !reflect.DeepEqual(got, want) {
		t.Fatalf("deepseek mcp:\n got %v\nwant %v", got, want)
	}

	empty := t.TempDir()
	if err := WriteAgentCLIMCPConfig(AgentCLIDeepSeek, HostAgentCLIHome(empty), map[string]any{"skip": map[string]any{}}, AgentCLIMCPOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := readAgentCLIMCPDoc(t, empty, rel); !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("deepseek empty mcp: %v", got)
	}
}

// Kimi defaults to a 60s MCP tool timeout, shorter than a write approval may
// wait; every server it gets carries the gateway's 10 minute budget.
func TestWriteAgentCLIMCPConfigKimiToolTimeout(t *testing.T) {
	home := t.TempDir()
	entries := map[string]any{
		"http":  map[string]any{"type": "http", "url": "http://127.0.0.1:1/mcp", "headers": map[string]any{"Authorization": "Bearer x"}},
		"stdio": map[string]any{"command": "node", "args": []string{"a.js"}},
	}
	if err := WriteAgentCLIMCPConfig(AgentCLIKimi, HostAgentCLIHome(home), entries, AgentCLIMCPOptions{}); err != nil {
		t.Fatal(err)
	}
	got := readAgentCLIMCPDoc(t, home, ".kimi-code/mcp.json").(map[string]any)["mcpServers"]
	want := normalizeJSON(t, map[string]any{
		"http":  map[string]any{"url": "http://127.0.0.1:1/mcp", "headers": map[string]any{"Authorization": "Bearer x"}, "toolTimeoutMs": 600000},
		"stdio": map[string]any{"command": "node", "args": []string{"a.js"}, "toolTimeoutMs": 600000},
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("kimi mcp:\n got %v\nwant %v", got, want)
	}
}

// Pi declares MCP tools to the model only when a server's exposure is
// "direct"; the default (codemode) leaves submit_gate_result invisible while
// the platform prompt tells the agent to call it natively.
func TestWriteAgentCLIMCPConfigPiExposesToolsDirectly(t *testing.T) {
	home := t.TempDir()
	entries := map[string]any{
		"agentflow-gate": map[string]any{
			"command":     ".agentflow/config/reviewer/gate-mcp",
			"env":         map[string]any{"AGENTFLOW_GATE_RESULT_PATH": ".agentflow/gate-results/review/attempt-1.json"},
			"description": "提交本节点的终态与理由",
		},
		"gateway": map[string]any{"url": "http://127.0.0.1:9/mcp", "headers": map[string]any{"Authorization": "Bearer x"}},
	}
	if err := WriteAgentCLIMCPConfig(AgentCLIPi, HostAgentCLIHome(home), entries, AgentCLIMCPOptions{}); err != nil {
		t.Fatal(err)
	}
	servers := readAgentCLIMCPDoc(t, home, agentCLIMCPPaths[AgentCLIPi]).(map[string]any)["mcpServers"].(map[string]any)
	for _, id := range []string{"agentflow-gate", "gateway"} {
		server, ok := servers[id].(map[string]any)
		if !ok {
			t.Fatalf("%s missing from pi mcp.json: %v", id, servers)
		}
		if server["exposure"] != "direct" {
			t.Fatalf("%s exposure = %v, want direct", id, server["exposure"])
		}
	}
	gate := servers["agentflow-gate"].(map[string]any)
	if gate["command"] != ".agentflow/config/reviewer/gate-mcp" || gate["description"] != "提交本节点的终态与理由" {
		t.Fatalf("gate entry lost its command or description: %v", gate)
	}
	if env, _ := gate["env"].(map[string]any); env["AGENTFLOW_GATE_RESULT_PATH"] == nil {
		t.Fatalf("gate entry lost its env: %v", gate)
	}

	// The same entries rendered for another CLI must stay exactly as before:
	// exposure and description are Pi-only keys.
	other := t.TempDir()
	if err := WriteAgentCLIMCPConfig(AgentCLIKimi, HostAgentCLIHome(other), entries, AgentCLIMCPOptions{}); err != nil {
		t.Fatal(err)
	}
	kimi := readAgentCLIMCPDoc(t, other, agentCLIMCPPaths[AgentCLIKimi]).(map[string]any)["mcpServers"].(map[string]any)
	for id, raw := range kimi {
		server := raw.(map[string]any)
		if _, ok := server["exposure"]; ok {
			t.Fatalf("%s leaked pi exposure into %s mcp.json: %v", id, AgentCLIKimi, server)
		}
		if _, ok := server["description"]; ok {
			t.Fatalf("%s leaked pi description into %s mcp.json: %v", id, AgentCLIKimi, server)
		}
	}
}
