package resp

import (
	"strings"
	"testing"
)

func TestReadRESP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PING", "*1\r\n$4\r\nPING\r\n", "PING"},
		{"SET", "*3\r\n$3\r\nSET\r\n$3\r\nkey1\r\n$6\r\nvalue1\r\n", "SET"},
		{"GET", "*2\r\n$3\r\nGET\r\n$3\r\nkey1\r\n", "GET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResp(strings.NewReader(tt.input))
			values, err := r.Read()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if values.Array[0].Bulk != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, values.Array[0].Bulk)
			}
		})
	}
}
