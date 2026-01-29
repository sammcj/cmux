// internal/workspace/morph_ports_fuzzing_test.go
package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// TestPortsFuzzingRandomPorts tests random port values
func TestPortsFuzzingRandomPorts(t *testing.T) {
	w := &Workspace{}

	// Add 1000 random ports
	for i := 0; i < 1000; i++ {
		port := rand.Intn(65536)
		w.AddActivePort(ActivePort{Port: port})
	}

	// All operations should work without panic
	_ = w.GetHTTPPorts()
	_ = w.GetPrimaryAppPort()

	for i := 0; i < 100; i++ {
		_ = w.GetActivePort(rand.Intn(65536))
	}
}

// TestPortsFuzzingRandomServices tests random service names
func TestPortsFuzzingRandomServices(t *testing.T) {
	w := &Workspace{}

	// Generate random service names
	for i := 0; i < 100; i++ {
		service := generateRandomString(rand.Intn(1000))
		w.AddActivePort(ActivePort{
			Port:    8000 + i,
			Service: service,
		})
	}

	// Verify all ports exist
	for i := 0; i < 100; i++ {
		port := w.GetActivePort(8000 + i)
		if port == nil {
			t.Errorf("Port %d not found", 8000+i)
		}
	}
}

// TestPortsFuzzingRandomContainers tests random container names
func TestPortsFuzzingRandomContainers(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 100; i++ {
		container := generateRandomString(rand.Intn(500))
		w.AddActivePort(ActivePort{
			Port:      8000 + i,
			Container: container,
		})
	}

	// Verify all ports exist
	for i := 0; i < 100; i++ {
		if w.GetActivePort(8000+i) == nil {
			t.Errorf("Port %d not found", 8000+i)
		}
	}
}

// TestPortsFuzzingRandomURLs tests random URL strings
func TestPortsFuzzingRandomURLs(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 100; i++ {
		url := "http://" + generateRandomString(rand.Intn(100)) + "/" + generateRandomString(rand.Intn(50))
		w.AddActivePort(ActivePort{
			Port: 8000 + i,
			URL:  url,
		})
	}

	// Verify all stored
	for i := 0; i < 100; i++ {
		if w.GetActivePort(8000+i) == nil {
			t.Errorf("Port %d not found", 8000+i)
		}
	}
}

// TestPortsFuzzingRandomProtocols tests random protocol values
func TestPortsFuzzingRandomProtocols(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 100; i++ {
		protocol := generateRandomString(rand.Intn(20))
		w.AddActivePort(ActivePort{
			Port:     8000 + i,
			Protocol: protocol,
		})
	}

	for i := 0; i < 100; i++ {
		if w.GetActivePort(8000+i) == nil {
			t.Errorf("Port %d not found", 8000+i)
		}
	}
}

// TestPortsFuzzingRandomOperations tests random sequences of operations
func TestPortsFuzzingRandomOperations(t *testing.T) {
	w := &Workspace{}

	operations := []func(){
		func() {
			w.AddActivePort(ActivePort{Port: rand.Intn(65536)})
		},
		func() {
			w.RemoveActivePort(rand.Intn(65536))
		},
		func() {
			w.GetActivePort(rand.Intn(65536))
		},
		func() {
			w.GetHTTPPorts()
		},
		func() {
			w.GetPrimaryAppPort()
		},
		func() {
			w.ClearActivePorts()
		},
	}

	// Execute 10000 random operations
	for i := 0; i < 10000; i++ {
		op := operations[rand.Intn(len(operations))]
		op()
	}
}

// TestPortsFuzzingIntBoundaries tests integer boundary values
func TestPortsFuzzingIntBoundaries(t *testing.T) {
	testCases := []struct {
		name      string
		port      int
		localPort int
	}{
		{"min_int", math.MinInt, 0},
		{"max_int", math.MaxInt, 0},
		{"zero", 0, 0},
		{"negative_one", -1, -1},
		{"max_port", 65535, 65535},
		{"over_max", 65536, 65536},
		{"large_negative", -1000000, -1000000},
		{"large_positive", 1000000, 1000000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:      tc.port,
				LocalPort: tc.localPort,
			})

			port := w.GetActivePort(tc.port)
			if port == nil {
				t.Error("Port not found")
			}
		})
	}
}

// TestPortsFuzzingBooleanSequences tests sequences of IsHTTP values
func TestPortsFuzzingBooleanSequences(t *testing.T) {
	w := &Workspace{}

	// Create pattern: true, true, false, true, true, false, ...
	for i := 0; i < 1000; i++ {
		isHTTP := (i%3 != 2)
		w.AddActivePort(ActivePort{
			Port:   8000 + i,
			IsHTTP: isHTTP,
		})
	}

	httpPorts := w.GetHTTPPorts()
	expectedHTTP := 666 + 1 // Two out of every three
	if len(httpPorts) != expectedHTTP {
		t.Errorf("Expected %d HTTP ports, got %d", expectedHTTP, len(httpPorts))
	}
}

// TestPortsFuzzingJSONRoundTrip tests JSON with random data
func TestPortsFuzzingJSONRoundTrip(t *testing.T) {
	w := &Workspace{ID: "ws-fuzz"}

	// Add random ports
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{
			Port:      rand.Intn(65536),
			Protocol:  generateRandomString(rand.Intn(10)),
			Service:   generateRandomString(rand.Intn(50)),
			Container: generateRandomString(rand.Intn(50)),
			LocalPort: rand.Intn(65536),
			URL:       generateRandomString(rand.Intn(100)),
			IsHTTP:    rand.Intn(2) == 0,
		})
	}

	// Serialize
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Deserialize
	var w2 Workspace
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Compare counts
	if len(w2.Morph.ActivePorts) != len(w.Morph.ActivePorts) {
		t.Errorf("Port count mismatch: %d vs %d",
			len(w.Morph.ActivePorts), len(w2.Morph.ActivePorts))
	}
}

// TestPortsFuzzingEmptyStrings tests various combinations of empty strings
func TestPortsFuzzingEmptyStrings(t *testing.T) {
	combinations := []ActivePort{
		{Port: 1, Protocol: "", Service: "", Container: "", URL: ""},
		{Port: 2, Protocol: "tcp", Service: "", Container: "", URL: ""},
		{Port: 3, Protocol: "", Service: "svc", Container: "", URL: ""},
		{Port: 4, Protocol: "", Service: "", Container: "cont", URL: ""},
		{Port: 5, Protocol: "", Service: "", Container: "", URL: "http://x"},
		{Port: 6, Protocol: "tcp", Service: "svc", Container: "", URL: ""},
		{Port: 7, Protocol: "tcp", Service: "", Container: "cont", URL: ""},
		{Port: 8, Protocol: "", Service: "svc", Container: "cont", URL: ""},
	}

	w := &Workspace{}
	for _, p := range combinations {
		w.AddActivePort(p)
	}

	if len(w.Morph.ActivePorts) != len(combinations) {
		t.Errorf("Expected %d ports, got %d", len(combinations), len(w.Morph.ActivePorts))
	}
}

// TestPortsFuzzingNullBytes tests null bytes in strings
func TestPortsFuzzingNullBytes(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{
		Port:      8080,
		Service:   "before\x00after",
		Container: "\x00start",
		URL:       "end\x00",
		Protocol:  "\x00\x00\x00",
	})

	port := w.GetActivePort(8080)
	if port.Service != "before\x00after" {
		t.Error("Null byte in service not preserved")
	}
}

// TestPortsFuzzingUnicodeCharacters tests various Unicode characters
func TestPortsFuzzingUnicodeCharacters(t *testing.T) {
	unicodeStrings := []string{
		"Hello世界",
		"Привет",
		"مرحبا",
		"שלום",
		"🚀🎉💻",
		"\\u0000",
		"Tab\tHere",
		"Line\nBreak",
		"Mix: 你好World🌍",
	}

	w := &Workspace{}
	for i, s := range unicodeStrings {
		w.AddActivePort(ActivePort{
			Port:    8000 + i,
			Service: s,
		})
	}

	// Verify all preserved
	for i, s := range unicodeStrings {
		port := w.GetActivePort(8000 + i)
		if port.Service != s {
			t.Errorf("Unicode string not preserved at index %d", i)
		}
	}
}

// TestPortsFuzzingControlCharacters tests control characters
func TestPortsFuzzingControlCharacters(t *testing.T) {
	controlChars := []string{
		"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07",
		"\x08", "\x09", "\x0A", "\x0B", "\x0C", "\x0D", "\x0E", "\x0F",
		"\x10", "\x11", "\x12", "\x13", "\x14", "\x15", "\x16", "\x17",
		"\x18", "\x19", "\x1A", "\x1B", "\x1C", "\x1D", "\x1E", "\x1F",
	}

	w := &Workspace{}
	for i, c := range controlChars {
		w.AddActivePort(ActivePort{
			Port:    8000 + i,
			Service: fmt.Sprintf("ctrl%s", c),
		})
	}

	// Verify all stored
	if len(w.Morph.ActivePorts) != len(controlChars) {
		t.Errorf("Expected %d ports, got %d", len(controlChars), len(w.Morph.ActivePorts))
	}
}

// TestPortsFuzzingVeryLongStrings tests very long strings
func TestPortsFuzzingVeryLongStrings(t *testing.T) {
	lengths := []int{100, 1000, 10000, 100000}

	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			w := &Workspace{}
			longStr := generateRandomString(length)
			w.AddActivePort(ActivePort{
				Port:    8080,
				Service: longStr,
			})

			port := w.GetActivePort(8080)
			if len(port.Service) != length {
				t.Errorf("Expected length %d, got %d", length, len(port.Service))
			}
		})
	}
}

// TestPortsFuzzingManyPorts tests adding many ports
func TestPortsFuzzingManyPorts(t *testing.T) {
	counts := []int{100, 1000, 10000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("count_%d", count), func(t *testing.T) {
			w := &Workspace{}
			for i := 0; i < count; i++ {
				w.AddActivePort(ActivePort{Port: i})
			}

			if len(w.Morph.ActivePorts) != count {
				t.Errorf("Expected %d ports, got %d", count, len(w.Morph.ActivePorts))
			}
		})
	}
}

// TestPortsFuzzingTimestampEdges tests timestamp edge cases
func TestPortsFuzzingTimestampEdges(t *testing.T) {
	testCases := []struct {
		name string
		time time.Time
	}{
		{"zero_time", time.Time{}},
		{"unix_epoch", time.Unix(0, 0)},
		{"far_future", time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)},
		{"negative_year", time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:         8080,
				DiscoveredAt: tc.time,
			})

			port := w.GetActivePort(8080)
			// If zero time was passed, AddActivePort sets it to now
			// Otherwise it should be preserved
			if !tc.time.IsZero() && !port.DiscoveredAt.Equal(tc.time) {
				t.Errorf("Timestamp not preserved: expected %v, got %v", tc.time, port.DiscoveredAt)
			}
		})
	}
}

// TestPortsFuzzingAllFieldsRandom tests all fields with random values
func TestPortsFuzzingAllFieldsRandom(t *testing.T) {
	for i := 0; i < 100; i++ {
		w := &Workspace{}
		port := ActivePort{
			Port:         rand.Intn(65536),
			Protocol:     generateRandomString(rand.Intn(20)),
			Service:      generateRandomString(rand.Intn(100)),
			Container:    generateRandomString(rand.Intn(100)),
			LocalPort:    rand.Intn(65536),
			URL:          generateRandomString(rand.Intn(200)),
			IsHTTP:       rand.Intn(2) == 0,
			DiscoveredAt: time.Now().Add(time.Duration(rand.Intn(1000000)) * time.Second),
		}

		w.AddActivePort(port)

		retrieved := w.GetActivePort(port.Port)
		if retrieved == nil {
			t.Errorf("Iteration %d: port not found", i)
		}
	}
}

// TestPortsFuzzingSpecialURLSchemes tests special URL schemes
func TestPortsFuzzingSpecialURLSchemes(t *testing.T) {
	schemes := []string{
		"http://", "https://", "ws://", "wss://", "ftp://",
		"file://", "tcp://", "udp://", "ssh://", "git://",
		"custom://", "myapp://", "x://", "://", "",
		"http:/", "http:", "http", "/",
	}

	w := &Workspace{}
	for i, scheme := range schemes {
		w.AddActivePort(ActivePort{
			Port: 8000 + i,
			URL:  scheme + "example.com",
		})
	}

	if len(w.Morph.ActivePorts) != len(schemes) {
		t.Errorf("Expected %d ports, got %d", len(schemes), len(w.Morph.ActivePorts))
	}
}

// TestPortsFuzzingJSONSpecialChars tests JSON special characters
func TestPortsFuzzingJSONSpecialChars(t *testing.T) {
	specialChars := []string{
		`"quoted"`,
		`back\slash`,
		`forward/slash`,
		`{braces}`,
		`[brackets]`,
		`key:value`,
		`comma,here`,
		`new
line`,
		`tab	here`,
		`mixed"quotes'and\escape`,
	}

	w := &Workspace{}
	for i, s := range specialChars {
		w.AddActivePort(ActivePort{
			Port:    8000 + i,
			Service: s,
		})
	}

	// JSON round-trip
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var w2 Workspace
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all preserved
	for i, s := range specialChars {
		port := w2.GetActivePort(8000 + i)
		if port.Service != s {
			t.Errorf("Special char string not preserved at index %d: expected %q, got %q", i, s, port.Service)
		}
	}
}

// TestPortsFuzzingRepeatedAddRemove tests repeated add/remove patterns
func TestPortsFuzzingRepeatedAddRemove(t *testing.T) {
	w := &Workspace{}

	// Pattern: add 10, remove 5, repeat
	for cycle := 0; cycle < 100; cycle++ {
		for i := 0; i < 10; i++ {
			w.AddActivePort(ActivePort{Port: 8000 + i})
		}
		for i := 0; i < 5; i++ {
			w.RemoveActivePort(8000 + i)
		}
	}

	// Should have 5 ports from last cycle
	if len(w.Morph.ActivePorts) != 5 {
		t.Errorf("Expected 5 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// Helper function to generate random strings
func generateRandomString(length int) string {
	if length <= 0 {
		return ""
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !@#$%^&*()_+-=[]{}|;':\",./<>?"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// TestPortsFuzzingBinaryData tests binary data in strings
func TestPortsFuzzingBinaryData(t *testing.T) {
	w := &Workspace{}

	// Create binary data
	binaryData := make([]byte, 256)
	for i := range binaryData {
		binaryData[i] = byte(i)
	}

	w.AddActivePort(ActivePort{
		Port:    8080,
		Service: string(binaryData),
	})

	port := w.GetActivePort(8080)
	if !bytes.Equal([]byte(port.Service), binaryData) {
		t.Error("Binary data not preserved")
	}
}
