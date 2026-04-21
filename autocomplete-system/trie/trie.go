package trie

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

type Value struct {
	Text      string
	frequency int
}

type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
	topK     []Value
}

type AutoComplete struct {
	Root *TrieNode
	k    int          // top k suggestion
	size int          // track the size of the Trie
	mu   sync.RWMutex // mutex used when swapping the old Trie with new Trie
}

// k = top K suggestions you want to set up
func NewAutoComplete(k int) *AutoComplete {
	return &AutoComplete{
		Root: &TrieNode{
			children: make(map[rune]*TrieNode),
			isEnd:    false,
		},
		k: k,
	}
}

func (ac *AutoComplete) Size() int {
	return ac.size
}

func (ac *AutoComplete) Search(prefix string) []Value {
	res := ac.traverse(prefix)
	if res == nil {
		return []Value{}
	}
	return res.topK
}
func (ac *AutoComplete) Swap(newAc *AutoComplete) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.Root = newAc.Root
}
func (ac *AutoComplete) Build(path string) error {
	// 1. Processes aggregated file to map
	processedData, err := buildFrequencyMap(path)
	if err != nil {
		return err
	}

	// 2. Build Trie based on processed K-V data
	for k, v := range processedData {
		ac.insert(k, v)
	}
	return nil
}

// walks through the trie and returns the node where the prefix ends
func (ac *AutoComplete) traverse(prefix string) *TrieNode {
	currentNode := ac.Root
	for _, c := range prefix {
		if _, exists := currentNode.children[c]; !exists {
			return nil
		}
		currentNode = currentNode.children[c]
	}
	return currentNode
}

func (ac *AutoComplete) insert(text string, count int) {
	currentNode := ac.Root
	traversedNodes := []*TrieNode{currentNode}
	for _, c := range text {
		if _, exists := currentNode.children[c]; !exists {
			newNode := &TrieNode{children: make(map[rune]*TrieNode)}
			currentNode.children[c] = newNode
			// currentNode.topK = make([]value, ac.k)
		}

		currentNode = currentNode.children[c]
		traversedNodes = append(traversedNodes, currentNode)
	}
	currentNode.isEnd = true
	ac.size++
	t := Value{Text: text, frequency: count}
	for _, node := range traversedNodes {
		node.topK = append(node.topK, t)
		sort.Slice(node.topK, func(i, j int) bool {
			return node.topK[i].frequency > node.topK[j].frequency
		})
		if len(node.topK) > ac.k {
			node.topK = node.topK[:ac.k]
		}
	}
}

// accepts file path, build and return K-V data
func buildFrequencyMap(path string) (map[string]int, error) {
	res := map[string]int{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		res[line]++

	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("faild to build: %s", scanner.Err())
	}
	return res, nil
}
