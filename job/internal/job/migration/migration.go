package migration

import (
	"os"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/job/migration/producer"

	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

type Migration struct {
	pattern string
}

func (migration *Migration) ExecuteJob(path string, tableName string) {
	// Create a wait group to wait for all workers to finish

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("***** STARTING MIGRTION JOB *****")
	var wg sync.WaitGroup
	var consumerWg sync.WaitGroup
	outputChunkDir := utils.GetChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		LOGGER.Error("Error reading directory:", err)
		return
	}

	for _, file := range files {

		wg.Add(1)
		go producer.Worker(outputChunkDir, file.Name(), &wg, &consumerWg, "")
	}

	// Wait for all workers to finish
	wg.Wait()
	consumerWg.Wait()
}

func (migration *Migration) GetFileNamePattern() string {
	return migration.pattern
}

func (migration *Migration) SetFileNamePattern(pattern string) {
	migration.pattern = pattern
}
