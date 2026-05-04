package util

import (
	"errors"
	"strings"
)

const (
	alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base     = uint64(len(alphabet)) // 62
)

var (
	ErrInvalidCode = errors.New("base62: invalid character in code")
	ErrEmptyCode   = errors.New("base62: code must not be empty")
)

// Encode converts a positive integer (e.g. a Postgres BIGSERIAL value)
// into a Base62 string. Encode(1) → "1", Encode(62) → "a0".
// The output length grows logarithmically; a 64-bit counter won't exceed
// 11 characters until you've created ~3.5 trillion URLs.
func Encode(n uint64) string {
	if n == 0 {
		return string(alphabet[0])
	}

	// Pre-allocate a small buffer; 11 chars covers the full uint64 range.
	buf := make([]byte, 0, 11)

	for n > 0 {
		buf = append(buf, alphabet[n%base])
		n /= base
	}

	// Digits were appended least-significant-first; reverse in place.
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}

	return string(buf)
}

// Decode converts a Base62 string back to its integer representation.
// Returns ErrEmptyCode or ErrInvalidCode on bad input.
func Decode(code string) (uint64, error) {
	if code == "" {
		return 0, ErrEmptyCode
	}

	var n uint64
	for _, ch := range code {
		idx := strings.IndexRune(alphabet, ch)
		if idx == -1 {
			return 0, ErrInvalidCode
		}
		n = n*base + uint64(idx)
	}

	return n, nil
}
