package logger

import (
	"fmt"
	"os"
	"sync"
)

// This records every query to the server, using append-only O(1)
type QueryLogger struct {
	mu   sync.Mutex
	file *os.File
}

func NewQueryLogger(path string) (*QueryLogger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &QueryLogger{file: f}, nil
}

func (ql *QueryLogger) Log(query string) error {
	ql.mu.Lock()
	defer ql.mu.Unlock()
	_, err := fmt.Fprintln(ql.file, query)
	return err
}
