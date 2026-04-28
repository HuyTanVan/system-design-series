package workers

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func AggregateData(inputPath, outputPath string) (bool, error) {
	start := time.Now()

	f, err := os.Open(inputPath)
	if err != nil {
		return false, fmt.Errorf("failed to open input file: %w", err)
	}
	defer f.Close()

	// 1. count frequencies using map
	freqMap := make(map[string]int)
	scanner := bufio.NewScanner(f)
	// increase buffer size for long lines
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		freqMap[line]++
	}

	if scanner.Err() != nil {
		return false, fmt.Errorf("failed to scan: %w", scanner.Err())
	}

	// 2. write to temp file first, rename on success (atomic write)
	tmpPath := outputPath + ".tmp"
	output, err := os.Create(tmpPath)
	if err != nil {
		return false, fmt.Errorf("failed to create output file: %w", err)
	}
	defer output.Close()

	for phrase, freq := range freqMap {
		fmt.Fprintf(output, "%s,%d\n", phrase, freq)
	}

	// atomic rename
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return false, fmt.Errorf("failed to rename temp file: %w", err)
	}

	log.Printf("data aggregated successfully | size=%d | time=%s", len(freqMap), time.Since(start))
	return true, nil
}

// // processes the txt file(for a specific kaggle dataset).
// // clean and convert txt to csv
// build log: 2026/04/24 19:30:22 raw txt file processed successfully | time=21m55.77968602s
// func ProcessTxtFile(path string) (bool, error) {
// 	start := time.Now()
// 	// 1. read the kaggle dataset txt file
// 	f, err := os.Open(path)
// 	if err != nil {
// 		return false, err
// 	}
// 	defer f.Close()
// 	output, err := os.Create("./data/cleaned-raw-data.txt")
// 	if err != nil {
// 		return false, err
// 	}
// 	defer output.Close()
// 	scanner := bufio.NewScanner(f)
// 	scanner.Scan() // skip the header line
// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())
// 		if line == "" {
// 			continue
// 		}

// 		// 2. extract, clean the data (convert to lowercase, etc.)
// 		// handle tab
// 		cols := strings.Split(line, "\t")
// 		if len(cols) < 2 {
// 			continue
// 		}
// 		// query is in the second column
// 		query := strings.TrimSpace(cols[1])
// 		if query == "" {
// 			continue
// 		}

// 		// convert to lowercase
// 		query = strings.ToLower(query)

// 		// 3. write the cleaned data to a new csv file
// 		fmt.Fprintln(output, query)

// 	}
// 	if scanner.Err() != nil {
// 		return false, fmt.Errorf("failed to build: %s", scanner.Err())
// 	}
// 	log.Printf("raw txt file processed successfully | time=%s", time.Since(start))
// 	return true, nil

// }
