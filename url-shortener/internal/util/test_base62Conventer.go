package util

import (
	"testing"
)

func TestEncode(t *testing.T) {
	cases := []struct {
		input uint64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "a"},         // first lowercase letter
		{35, "z"},         // last lowercase letter
		{36, "A"},         // first uppercase letter
		{61, "Z"},         // last uppercase letter
		{62, "10"},        // first two-char code
		{3844, "100"},     // 62^2
		{1000000, "4c92"}, // arbitrary mid-range value
	}

	for _, tc := range cases {
		got := Encode(tc.input)
		if got != tc.want {
			t.Errorf("Encode(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDecode(t *testing.T) {
	cases := []struct {
		input string
		want  uint64
		isErr bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"Z", 61, false},
		{"10", 62, false},
		{"100", 3844, false},
		{"4c92", 1000000, false},
		{"", 0, true},   // ErrEmptyCode
		{"!!", 0, true}, // ErrInvalidCode
	}

	for _, tc := range cases {
		got, err := Decode(tc.input)
		if tc.isErr {
			if err == nil {
				t.Errorf("Decode(%q) expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("Decode(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Decode(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// TestRoundTrip verifies Encode → Decode is a perfect inverse for a
// representative sample of values including edge cases and large numbers.
func TestRoundTrip(t *testing.T) {
	inputs := []uint64{
		0, 1, 61, 62, 63, 3843, 3844, 1_000_000,
		999_999_999, 1<<32 - 1, 1<<48 - 1, 1<<62 - 1,
	}

	for _, n := range inputs {
		code := Encode(n)
		got, err := Decode(code)
		if err != nil {
			t.Errorf("Decode(Encode(%d)) error: %v", n, err)
			continue
		}
		if got != n {
			t.Errorf("RoundTrip(%d): got %d via code %q", n, got, code)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Encode(uint64(i))
	}
}

func BenchmarkDecode(b *testing.B) {
	code := Encode(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(code) //nolint:errcheck
	}
}
