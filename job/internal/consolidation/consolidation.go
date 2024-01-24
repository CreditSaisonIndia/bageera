package consolidation

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

func Consolidate() (string, error) {
	//"/Users/taylor/workspace/go/offer/PSB/file_PSB_500000/chunks"
	LOGGER := customLogger.GetLogger()
	chunksDir := utils.GetChunksDir()
	LOGGER.Info("chunksDir :", chunksDir)
	fileNameWithoutExt, _ := utils.GetFileName()
	fileNameWithoutExt = fileNameWithoutExt + "_valid"
	LOGGER.Info("fileName :", fileNameWithoutExt)
	// Retrieve only original CSV files in the chunks directory
	s_f_dir := filepath.Join(chunksDir, fileNameWithoutExt+"_*_*.csv")
	LOGGER.Info("s_f_dir :", s_f_dir)
	matches, err := filepath.Glob(s_f_dir)
	if err != nil {
		LOGGER.Error("Error filepath.Glob: ", err)
		return "", err
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

		failureFilePath := fmt.Sprintf("%s_%s_failure.csv", fileNameWithoutExt, chunkID)
		successFilePath := fmt.Sprintf("%s_%s_success.csv", fileNameWithoutExt, chunkID)
		failureRowCount, err := getRowCount(filepath.Join(chunksDir, failureFilePath))
		if err != nil {
			LOGGER.Error("Error failureRowCount getRowCount: ", err)
			return "", err
		}
		failureCounts[chunkID] = failureRowCount

		successRowCount, err := getRowCount(filepath.Join(chunksDir, successFilePath))
		if err != nil {
			LOGGER.Error("Error successRowCount getRowCount: ", err)
			return "", err
		}
		successCounts[chunkID] = successRowCount
	}
	resultDir := utils.GetResultsDir()
	if err := os.MkdirAll(resultDir, os.ModePerm); err != nil {
		LOGGER.Error("Error MkdirAll result dir : ", err)
		return "", err
	}
	rowCountPath := filepath.Join(resultDir, "row_counts.csv")
	log.Println("rowCountPath : ", rowCountPath)
	// Create a CSV file for counts
	csvFile, err := os.Create(rowCountPath)
	if err != nil {
		LOGGER.Error("Error os.Create: ", err)
		return "", err
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	// Write CSV header
	header := []string{"ChunkID", "FileName", "FailureCount", "SuccessCount"}
	if err := csvWriter.Write(header); err != nil {
		LOGGER.Error("Error csvWriter.Write(header): ", err)
		return "", err
	}

	// Iterate over processed chunks and write counts to CSV
	for chunkID, _ := range processedChunks {
		// Write row counts to CSV
		row := []string{chunkID, fmt.Sprintf("%s_%s.csv", fileNameWithoutExt, chunkID),
			fmt.Sprintf("%d", failureCounts[chunkID]), fmt.Sprintf("%d", successCounts[chunkID])}
		if err := csvWriter.Write(row); err != nil {
			LOGGER.Error("Error csvWriter.Write(row): ", err)
			return "", err
		}
	}

	LOGGER.Info("Row counts written to row_counts.csv")
	return rowCountPath, nil
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

type ValidationResult struct {
	IsValid bool
	Err     error
}
type VerifyConsolidator interface {
	CheckConsolidator(filePath string) ValidationResult
}
type VerifyConsolidatorImpl struct{}

func (v *VerifyConsolidatorImpl) CheckConsolidator(filePath string) ValidationResult {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return ValidationResult{false, err}
	}
	defer file.Close()
	// Create a CSV reader
	reader := csv.NewReader(file)
	// Read all records from the CSV file
	records, err := reader.ReadAll()
	if err != nil {
		return ValidationResult{false, err}
	}
	// Task 1: Check if there is data apart from the header
	if len(records) <= 1 {
		return ValidationResult{false, err}
	}
	// Task 2: Check if all FailureCount rows have only one value
	for _, record := range records[1:] { // Start from index 1 to skip the header
		failureCount, err := strconv.Atoi(record[2])
		if err != nil {
			return ValidationResult{false, err}
		}
		if failureCount != 1 {
			return ValidationResult{false, err}
		}
	}
	// All checks passed
	return ValidationResult{true, err}
}
