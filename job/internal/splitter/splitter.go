package splitter

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	splitCsv "github.com/tolik505/split-csv"
	"go.uber.org/zap"
)

var LOGGER *zap.SugaredLogger

func SplitCsv() {

	LOGGER := customLogger.GetLogger()
	outputChunkDir := utils.GetChunksDir()

	if err := os.MkdirAll(outputChunkDir, os.ModePerm); err != nil {
		LOGGER.Info("Error creating directory:", err)
		return
	}

	splitter := splitCsv.New()
	splitter.Separator = ";"          // "," is by default
	splitter.FileChunkSize = 10000000 //in bytes (200MB)
	baseDir := utils.GetBaseDir()
	fileNameWithoutExt, fileName := utils.GetFileName()
	inputPath := filepath.Join(baseDir, fileName)
	ErrBigFileChunkSize := errors.New("file chunk size is bigger than input file")
	_, err := splitter.Split(inputPath, outputChunkDir)
	if err != nil && ErrBigFileChunkSize.Error() == err.Error() {

		LOGGER.Info("Chunking Failed with message : ", err, "  |  Hence creating a chunk with same file")
		LOGGER.Info("fileName : ", fileName)
		LOGGER.Info("fileNameWithoutExt : ", fileNameWithoutExt)
		LOGGER.Info("inputPath : ", inputPath)
		LOGGER.Info("outputChunkDir : ", outputChunkDir)
		fileUtilityWrapper.Copy(fileName, fileNameWithoutExt, inputPath, outputChunkDir)
	} else if err != nil {
		LOGGER.Info("Error While Splitting :", err)
		return
	}
	csvCount, err := countCSVFiles(outputChunkDir)
	if err != nil {
		LOGGER.Info("Error counting CSV files:", err)
		return
	}

	log.Printf("Total CSV files in %s: %d\n", outputChunkDir, csvCount)

}

func countCSVFiles(directory string) (int, error) {
	files, err := filepath.Glob(filepath.Join(directory, "*.csv"))
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
