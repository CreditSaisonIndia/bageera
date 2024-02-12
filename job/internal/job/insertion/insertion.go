package insertion

import (
	"os"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion/producer"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

type Insertion struct {
}

func (insertion *Insertion) Execute(path string) {
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

	for _, file := range files {
		wg.Add(1)
		go producer.Worker(outputChunkDir, file.Name(), &wg, &consumerWg)
	}

	// Wait for all workers to finish
	wg.Wait()
	consumerWg.Wait()
}
