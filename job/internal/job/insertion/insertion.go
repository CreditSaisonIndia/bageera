package insertion

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion/producer"

	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

type Insertion struct {
	pattern string
}

func (insertion *Insertion) ExecuteJob(path string, tableName string) {
	// Create a wait group to wait for all workers to finish

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("***** STARTING INSERTION JOB *****")
	var wg sync.WaitGroup
	var consumerWg sync.WaitGroup
	outputChunkDir := utils.GetChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		LOGGER.Error("Error reading directory:", err)
		return
	}

	pattern := insertion.pattern
	regex := regexp.MustCompile(pattern)
	for _, file := range files {
		if file.IsDir() {
			// Get the subdirectory path
			subDir := filepath.Join(outputChunkDir, file.Name())

			subDirFiles, err := os.ReadDir(subDir)
			if err != nil {
				LOGGER.Error("Error reading directory:", err)
				return
			}

			for _, file := range subDirFiles {
				if file.Type().IsRegular() && regex.MatchString(file.Name()) {
					wg.Add(1)
					go producer.Worker(subDir, file.Name(), &wg, &consumerWg, tableName)
				}
			}

		}

	}

	// Wait for all workers to finish
	wg.Wait()
	consumerWg.Wait()
}

func (insertion *Insertion) GetFileNamePattern() string {
	return insertion.pattern
}

func (insertion *Insertion) SetFileNamePattern(pattern string) {
	insertion.pattern = pattern
}
