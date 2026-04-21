package trie

import (
	"testing"
)

func BenchmarkSearch(b *testing.B) {
	ac := NewAutoComplete(5)
	ac.Build("../logger/query-log.txt")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ac.Search("how to cook")
	}
}
