package consolidation

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

func Consolidate() {
	//"/Users/taylor/workspace/go/offer/PSB/file_PSB_500000/chunks"

	chunksDir := utils.GetChunksDir()
	fileName, _ := utils.GetFileName()
	// Retrieve only original CSV files in the chunks directory
	s_f_dir := filepath.Join(chunksDir, fileName+"_*_*.csv")

	matches, err := filepath.Glob(s_f_dir)
	if err != nil {
		log.Fatal(err)
	}

	// Create maps to track processed chunks and their counts
	processedChunks := make(map[string]bool)
	failureCounts := make(map[string]int)
	successCounts := make(map[string]int)

	// Iterate over original CSV files and count rows for failure and success files
	for _, filePath := range matches {
		chunkID := extractChunkID(filePath)

		// Skip processing if the chunk has already been processed
		if processedChunks[chunkID] {
			continue
		}
		processedChunks[chunkID] = true

		failureFilePath := fmt.Sprintf("%s_%s_failure.csv", fileName, chunkID)
		successFilePath := fmt.Sprintf("%s_%s_success.csv", fileName, chunkID)
		failureRowCount, err := getRowCount(filepath.Join(chunksDir, failureFilePath))
		if err != nil {
			log.Fatal(err)
		}
		failureCounts[chunkID] = failureRowCount

		successRowCount, err := getRowCount(filepath.Join(chunksDir, successFilePath))
		if err != nil {
			log.Fatal(err)
		}
		successCounts[chunkID] = successRowCount
	}
	resultDir := utils.GetResultsDir()
	if err := os.MkdirAll(resultDir, os.ModePerm); err != nil {
		log.Println("Error creating directory:", err)
		return
	}
	rowCountPath := filepath.Join(resultDir, "row_counts.csv")
	log.Println("rowCountPath : ", rowCountPath)
	// Create a CSV file for counts
	csvFile, err := os.Create(rowCountPath)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	// Write CSV header
	header := []string{"ChunkID", "FileName", "FailureCount", "SuccessCount"}
	if err := csvWriter.Write(header); err != nil {
		log.Fatal(err)
	}

	// Iterate over processed chunks and write counts to CSV
	for chunkID, _ := range processedChunks {
		// Write row counts to CSV
		row := []string{chunkID, fmt.Sprintf("%s_%s.csv", fileName, chunkID),
			fmt.Sprintf("%d", failureCounts[chunkID]), fmt.Sprintf("%d", successCounts[chunkID])}
		if err := csvWriter.Write(row); err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Row counts written to row_counts.csv")
}

func getRowCount(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	rowCount := 0

	// Assume the first line is the header and skip it
	if scanner.Scan() {
		rowCount++
	}

	for scanner.Scan() {
		rowCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return rowCount, nil
}

func extractChunkID(filePath string) string {
	// Assuming the file name format is "file_PSB_500000_{chunkID}.csv"
	fileName := filepath.Base(filePath)
	parts := strings.Split(fileName, "_")
	chunkID := parts[len(parts)-2]
	return chunkID
}
