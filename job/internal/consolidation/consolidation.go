package consolidation

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

func Consolidate(failurePattern *regexp.Regexp, successPattern *regexp.Regexp, failureFilePathFormat, successFilePathFormat, fileNameWithoutExt, resultFileName string, afterLen int) (string, error) {
	//"/Users/taylor/workspace/go/offer/PSB/file_PSB_500000/chunks"
	LOGGER := customLogger.GetLogger()
	// Retrieve only original CSV files in the chunks directory

	outputChunkDir := utils.GetChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		LOGGER.Error("Error reading directory:", err)
		return "", err
	}
	resultDir := utils.GetResultsDir()
	if err := os.MkdirAll(resultDir, os.ModePerm); err != nil {
		LOGGER.Error("Error MkdirAll result dir : ", err)
		return "", err
	}
	rowCountPath := filepath.Join(resultDir, resultFileName)
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

	for _, file := range files {
		if file.IsDir() {
			// Get the subdirectory path
			subDir := filepath.Join(outputChunkDir, file.Name())

			subDirFiles, err := os.ReadDir(subDir)
			if err != nil {
				LOGGER.Error("Error reading directory:", err)
				return "", err
			}
			// Create maps to track processed chunks and their counts
			processedChunks := make(map[string]bool)
			failureCounts := make(map[string]int)
			successCounts := make(map[string]int)
			for _, file := range subDirFiles {

				LOGGER.Info("file :", file.Name())
				if err != nil {
					LOGGER.Error("Error filepath.Glob: ", err)
					return "", err
				}

				var chunkID string
				if !processedChunks[chunkID] && (failurePattern.MatchString(file.Name()) || successPattern.MatchString(filepath.Base(file.Name()))) {
					LOGGER.Info("Processing File  : ", file.Name())
					chunkID := extractChunkID(file.Name(), afterLen)
					// Skip processing if the chunk has already been processed

					failureFilePath := fmt.Sprintf(failureFilePathFormat, fileNameWithoutExt, chunkID)
					successFilePath := fmt.Sprintf(successFilePathFormat, fileNameWithoutExt, chunkID)
					failureRowCount, err := getRowCount(filepath.Join(subDir, failureFilePath))
					LOGGER.Info("SUCCESS PATH : ", successFilePath)
					LOGGER.Info("FAILURE PATH : ", failureFilePath)
					if err != nil {
						LOGGER.Error("Error failureRowCount getRowCount: ", err)
						return "", err
					}
					failureCounts[chunkID] = failureRowCount

					successRowCount, err := getRowCount(filepath.Join(subDir, successFilePath))
					if err != nil {
						LOGGER.Error("Error successRowCount getRowCount: ", err)
						return "", err
					}
					successCounts[chunkID] = successRowCount
					processedChunks[chunkID] = true
					LOGGER.Info("Processed File  : ", file.Name())

				}

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

		}

	}

	LOGGER.Info("Row counts written to -----> ", resultFileName)
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

func extractChunkID(filePath string, afterLen int) string {
	// Assuming the file name format is "file_PSB_1000000_1_valid_1000000_exist_success.csv"
	fileName := filepath.Base(filePath)
	parts := strings.Split(fileName, "_")
	chunkID := parts[len(parts)-afterLen]
	return chunkID
}

type ResultInterface interface {
	show()
}

type CommonFields struct {
	Err error
}

type VerifyCountResult struct {
	CommonFields
	IsValid bool
}

func (v *VerifyCountResult) show() {
	fmt.Printf("show")
}

type ExistenceValidationResult struct {
	CommonFields
	AllPresent  bool
	AllAbsent   bool
	SomePresent bool
}

func (e *ExistenceValidationResult) show() {
	fmt.Printf("show")
}

type ValidationResult struct {
	IsValid     bool
	AllPresent  bool
	AllAbsent   bool
	SomePresent bool
	Err         error
}

type VerifyConsolidator interface {
	VerifyCount(filePath string) ValidationResult
}
type JobVerifyConsolidatorImpl struct{}

func (v *JobVerifyConsolidatorImpl) VerifyCount(filePath string) ResultInterface {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return &VerifyCountResult{
			CommonFields: CommonFields{Err: err},
			IsValid:      false,
		}
	}
	defer file.Close()
	// Create a CSV reader
	reader := csv.NewReader(file)
	// Read all records from the CSV file
	records, err := reader.ReadAll()
	if err != nil {
		return &VerifyCountResult{
			CommonFields: CommonFields{Err: err},
			IsValid:      false,
		}
	}
	// Task 1: Check if there is data apart from the header
	if len(records) <= 1 {
		return &VerifyCountResult{
			CommonFields: CommonFields{Err: err},
			IsValid:      false,
		}
	}
	// Task 2: Check if all FailureCount rows have only one value
	for _, record := range records[1:] { // Start from index 1 to skip the header
		failureCount, err := strconv.Atoi(record[2])
		if err != nil {
			return &VerifyCountResult{
				CommonFields: CommonFields{Err: err},
				IsValid:      false,
			}
		}
		if failureCount != 1 {
			return &VerifyCountResult{
				CommonFields: CommonFields{Err: err},
				IsValid:      false,
			}
		}
	}
	// All checks passed
	return &VerifyCountResult{
		CommonFields: CommonFields{Err: nil},
		IsValid:      true,
	}
}

type ExistenceVerifyConsolidatorImpl struct{}

func (v *ExistenceVerifyConsolidatorImpl) VerifyCount(filePath string) ResultInterface {
	// Open the CSV file
	var allPresent, allAbsent, somePresent bool = false, false, false
	file, err := os.Open(filePath)
	if err != nil {
		return &ExistenceValidationResult{
			CommonFields: CommonFields{Err: err},
			AllPresent:   false,
			AllAbsent:    false,
			SomePresent:  false,
		}

	}
	defer file.Close()
	// Create a CSV reader
	reader := csv.NewReader(file)
	// Read all records from the CSV file
	records, err := reader.ReadAll()
	if err != nil {
		return &ExistenceValidationResult{
			CommonFields: CommonFields{Err: err},
			AllPresent:   false,
			AllAbsent:    false,
			SomePresent:  false,
		}
	}
	// Task 1: Check if there is data apart from the header
	countRecordsLength := len(records)
	if countRecordsLength <= 1 {
		return &ExistenceValidationResult{
			CommonFields: CommonFields{Err: err},
			AllPresent:   false,
			AllAbsent:    false,
			SomePresent:  false,
		}
	}
	// Task 2: Check if all FailureCount rows have only one value
	var localFailureCount, localSuccessCount int

	for _, record := range records[1:] { // Start from index 1 to skip the header
		failureCount, err := strconv.Atoi(record[2])
		localFailureCount += failureCount
		successCount, err := strconv.Atoi(record[3])
		localSuccessCount += successCount
		if err != nil {
			return &ExistenceValidationResult{
				CommonFields: CommonFields{Err: err},
				AllPresent:   false,
				AllAbsent:    false,
				SomePresent:  false,
			}
		}
	}
	//All Counts includes header

	//overAllRecords := localFailureCount + localSuccessCount

	if localFailureCount == countRecordsLength-1 {
		allPresent = true
	} else if localSuccessCount == countRecordsLength-1 {
		allAbsent = true
	} else {
		somePresent = true
	}

	// All checks passed
	return &ExistenceValidationResult{
		CommonFields: CommonFields{Err: err},
		AllPresent:   allPresent,
		AllAbsent:    allAbsent,
		SomePresent:  somePresent,
	}
}
