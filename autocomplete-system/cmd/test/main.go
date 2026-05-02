package main

import (
	"autocomplete/internal/trie"
	"fmt"
)

func main() {
	ac := trie.NewAutoComplete(1)
	// insert some test data

	ac = ac.FakeBuild(map[string]int{
		"x": 1,
	})

	fmt.Printf("Node size: %d bytes\n", ac.GetNodeSize("x"))
	fmt.Printf("Node size: %d bytes\n", ac.GetNodeSize("y"))

	maxS := ac.TrieSize()
	fmt.Printf("Trie size: %d bytes\n", maxS)

	fmt.Println(ac.Search("x"))
}
