package deletion

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/job/deletion/producer"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

type Deletion struct {
	pattern string
}

func (d *Deletion) ExecuteJob(path string, tableName string) {
	// Create a wait group to wait for all workers to finish

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("***** STARTING DELETION JOB *****")
	var wg sync.WaitGroup
	var consumerWg sync.WaitGroup
	outputChunkDir := utils.GetChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		LOGGER.Error("Error reading directory:", err)
		return
	}

	pattern := d.pattern
	regex := regexp.MustCompile(pattern)
	// 1,2,3,4,5,6,...
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

func (deletion *Deletion) GetFileNamePattern() string {
	return deletion.pattern
}

func (updation *Deletion) SetFileNamePattern(pattern string) {
	updation.pattern = pattern
}
