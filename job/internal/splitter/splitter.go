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

func SplitCsv() error {

	LOGGER := customLogger.GetLogger()
	outputChunkDir := utils.GetChunksDir()

	if err := os.MkdirAll(outputChunkDir, os.ModePerm); err != nil {
		LOGGER.Info("Error creating directory:", err)
		return err
	}

	splitter := splitCsv.New()
	splitter.Separator = ";"          // "," is by default
	splitter.FileChunkSize = 10000000 //in bytes (200MB)
	baseDir := utils.GetMetadataBaseDir()
	fileNameWithoutExt, fileName := utils.GetFileName()
	fileNameWithoutExt = fileNameWithoutExt + "_valid"
	inputPath := filepath.Join(baseDir, fileNameWithoutExt+".csv")
	ErrBigFileChunkSize := errors.New("file chunk size is bigger than input file")
	_, err := splitter.Split(inputPath, outputChunkDir)
	if err != nil && ErrBigFileChunkSize.Error() == err.Error() {

		LOGGER.Info("Chunking Failed with message : ", err, "  |  Hence creating a chunk with same file")
		LOGGER.Info("fileName : ", fileName)
		LOGGER.Info("fileNameWithoutExt : ", fileNameWithoutExt)
		LOGGER.Info("inputPath : ", inputPath)
		LOGGER.Info("outputChunkDir : ", outputChunkDir)
		err = fileUtilityWrapper.Copy(fileName, fileNameWithoutExt, inputPath, outputChunkDir)
		if err != nil {
			LOGGER.Error("Error While Copying  :", err)
			return err
		}
	} else if err != nil {
		LOGGER.Error("Error While Splitting :", err)
		return err
	}
	csvCount, err := countCSVFiles(outputChunkDir)
	if err != nil {
		LOGGER.Error("Error counting CSV files:", err)
		return err
	}

	log.Printf("Total CSV files in %s: %d\n", outputChunkDir, csvCount)
	return nil
}

func countCSVFiles(directory string) (int, error) {
	files, err := filepath.Glob(filepath.Join(directory, "*.csv"))
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
