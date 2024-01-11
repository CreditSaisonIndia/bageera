package insertion

import (
	"log"
	"os"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/job/insertion/producer"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

func BeginInsertion() {
	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup
	var consumerWg sync.WaitGroup
	outputChunkDir := utils.GetChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		log.Println("Error reading directory:", err)
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
