package trie

import (
	"encoding/csv"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// notes on memory usage:
// string header: 16 bytes (ptr + len, always)
// string data: len(text) = byte
// int: frequency = 8 bytes
// estimate minimum: 16 + 0(empty str) + 8 = 24 bytes
type Value struct {
	Text      string
	Frequency int
}

// notes on memory usage:
// map header: 8 bytes (poiner, always)
// Children: len(Children) * (rune + pointer) = len(Children) * (4 + 8) bytes
// IsEnd: 1 byte + 7 bytes(padding) = 8 bytes
// TopK Slice header: 24 bytes (pointer, len, cap, always)
// TopK data: len(TopK) * (Value)
// estimate minimum: 8 + 8 + 24 = 40 bytes (without children and TopK empty)
type TrieNode struct {
	Children map[rune]*TrieNode
	IsEnd    bool
	TopK     []Value
}

type AutoComplete struct {
	Root     *TrieNode
	k        int          // top k suggestion
	size     int          // track the size of the Trie
	mu       sync.RWMutex // mutex used when swapping the old Trie with new Trie
	numNodes int          // track the number of nodes in the Trie
}

// k = top K suggestions you want to set up
func NewAutoComplete(k int) *AutoComplete {
	return &AutoComplete{
		Root: &TrieNode{
			Children: make(map[rune]*TrieNode),
			IsEnd:    false,
		},
		k: k,
	}
}

func (node *TrieNode) deepSize() uintptr {
	if node == nil {
		return 0
	}

	size := node.size()

	for _, child := range node.Children {
		size += child.deepSize()
	}

	return size
}

func (node *TrieNode) size() uintptr {
	if node == nil {
		return 0
	}

	// struct shell
	size := unsafe.Sizeof(*node)

	// Children map data: each entry = rune (4 bytes) + pointer (8 bytes)
	size += uintptr(len(node.Children)) * (unsafe.Sizeof(rune(0)) + unsafe.Sizeof((*TrieNode)(nil)))

	// TopK slice data: each Value = 16 (string header) + len(text) (string data) + 8 (frequency)
	for _, v := range node.TopK {
		size += 16 + uintptr(len(v.Text)) + 8
	}

	return size
}

func (ac *AutoComplete) TrieSize() uintptr {
	return ac.Root.deepSize()
}

// public — takes prefix string
func (ac *AutoComplete) GetNodeSize(prefix string) uintptr {
	node := ac.traverse(prefix)
	return node.size()
}

func (ac *AutoComplete) Size() int {
	return ac.size
}

func (ac *AutoComplete) GetNumNodes() int {
	return ac.numNodes
}

func (ac *AutoComplete) K() int {
	return ac.k
}

func (ac *AutoComplete) Search(prefix string) []Value {
	res := ac.traverse(prefix)
	if res == nil {
		return []Value{}
	}
	return res.TopK
}

func (ac *AutoComplete) Swap(newAc *AutoComplete) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.Root = newAc.Root
	ac.size = newAc.size
}

func (ac *AutoComplete) FakeBuild(data map[string]int) *AutoComplete {
	fakeAc := NewAutoComplete(ac.k)
	for text, frequency := range data {
		fakeAc.insert(text, frequency)
	}
	return fakeAc
}

// build from a processed csv file
func (ac *AutoComplete) Build(path string) error {
	// 1. Processes aggregated file to map
	processedData, err := buildFrequencyMap(path)
	if err != nil {
		return err
	}

	// 2. Build Trie based on processed K-V data
	for k, v := range processedData {
		// if v < 5 {
		// 	continue
		// }
		ac.insert(k, v)
	}
	return nil
}

// walks through the trie and returns the node where the prefix ends
func (ac *AutoComplete) traverse(prefix string) *TrieNode {
	currentNode := ac.Root
	for _, c := range prefix {
		if _, exists := currentNode.Children[c]; !exists {
			return nil
		}
		currentNode = currentNode.Children[c]
	}
	return currentNode
}

func (ac *AutoComplete) insert(text string, count int) {
	currentNode := ac.Root
	traversedNodes := []*TrieNode{currentNode}
	for _, c := range text {
		if _, exists := currentNode.Children[c]; !exists {
			newNode := &TrieNode{Children: make(map[rune]*TrieNode)}
			currentNode.Children[c] = newNode
			// currentNode.TopK = make([]Value, ac.k)
			ac.numNodes++

		}

		currentNode = currentNode.Children[c]
		traversedNodes = append(traversedNodes, currentNode)
	}
	currentNode.IsEnd = true
	ac.size++
	t := Value{Text: text, Frequency: count}
	for _, node := range traversedNodes {
		node.TopK = append(node.TopK, t)
		sort.Slice(node.TopK, func(i, j int) bool {
			return node.TopK[i].Frequency > node.TopK[j].Frequency
		})
		if len(node.TopK) > ac.k {
			node.TopK = node.TopK[:ac.k]
		}
	}
}

// accepts csv path, build and return K-V data
func buildFrequencyMap(path string) (map[string]int, error) {
	res := make(map[string]int)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.ReuseRecord = true

	var recordsProcessed int
	var parseErrors int

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			parseErrors++
			continue
		}

		if len(record) < 2 {
			continue
		}

		key := strings.TrimSpace(record[0])
		valStr := strings.TrimSpace(record[1])

		val, err := strconv.Atoi(valStr)
		if err != nil {
			parseErrors++
			continue
		}

		res[key] += val
		recordsProcessed++
	}

	return res, nil
}

// // accepts txt path, build and return K-V data
// func buildFrequencyMap(path string) (map[string]int, error) {
// 	res := map[string]int{}
// 	f, err := os.Open(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()

// 	scanner := bufio.NewScanner(f)
// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())
// 		if line == "" {
// 			continue
// 		}

// 		res[line]++

// 	}
// 	if scanner.Err() != nil {
// 		return nil, fmt.Errorf("faild to build: %s", scanner.Err())
// 	}
// 	return res, nil
// }
