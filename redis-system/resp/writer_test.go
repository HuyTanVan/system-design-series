package resp

import (
	"strings"
	"testing"
)

func TestWriteRESP(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		{"simple string", Value{Typ: "string", Str: "OK"}, "+OK\r\n"},
		{"error", Value{Typ: "error", Str: "ERR"}, "-ERR\r\n"},
		{"bulk string", Value{Typ: "bulk", Bulk: "PONG"}, "$4\r\nPONG\r\n"},
		// {"null", Value{Typ: "null"}, "$-1\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			w := NewWriter(&buf)
			w.Write(tt.input)
			if buf.String() != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}
