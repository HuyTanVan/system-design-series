// serializer.go contains functions to serialize and deserialize the trie data structure to and from disk using gob encoding.
// This allows us to persist the state of the autocomplete system across restarts and share it between different worker instances if needed.
package workers

import (
	"autocomplete/internal/trie"
	"encoding/gob"
	"fmt"
	"os"
)

func SerializeTrie(ac *trie.AutoComplete, outputPath string) error {
	tmpPath := outputPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}

	encoder := gob.NewEncoder(f)
	if err := encoder.Encode(ac.Root); err != nil {
		f.Close()
		return fmt.Errorf("failed to encode trie: %w", err)
	}

	// close BEFORE rename
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func DeserializeTrie(inputPath string, k int) (*trie.AutoComplete, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer f.Close()

	ac := trie.NewAutoComplete(k)
	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(ac.Root); err != nil {
		return nil, fmt.Errorf("failed to decode trie: %w", err)
	}

	return ac, nil
}
