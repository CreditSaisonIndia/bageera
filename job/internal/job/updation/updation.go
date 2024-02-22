package updation

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/job/updation/producer"

	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

type Updation struct {
	pattern string
}

func (updation *Updation) ExecuteJob(path string, tableName string) {
	// Create a wait group to wait for all workers to finish

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("***** STARTING UPDATION JOB *****")
	var wg sync.WaitGroup
	var consumerWg sync.WaitGroup
	outputChunkDir := utils.GetChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		LOGGER.Error("Error reading directory:", err)
		return
	}

	pattern := updation.pattern
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
					go producer.Worker(subDir, file.Name(), &wg, &consumerWg)
				}
			}

		}

	}

	// Wait for all workers to finish
	wg.Wait()
	consumerWg.Wait()
}

func (updation *Updation) GetFileNamePattern() string {
	return updation.pattern
}

func (updation *Updation) SetFileNamePattern(pattern string) {
	updation.pattern = pattern
}
