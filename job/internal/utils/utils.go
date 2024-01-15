package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

func GetBaseDir() string {
	fmt.Printf("Fetching EFS base path value From Settings : %s\n", serviceConfig.ApplicationSetting.EfsBasePath)
	fmt.Printf("Fetching EFS base path value From env : %s\n", os.Getenv("efsBasePath"))
	efsBasePath := os.Getenv("efsBasePath")
	objectKey := serviceConfig.ApplicationSetting.ObjectKey
	dirPath := filepath.Dir(objectKey)
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return filepath.Join(efsBasePath, dirPath, fileNameWithoutExt)

}

func GetChunksDir() string {
	return filepath.Join(GetBaseDir(), "chunks")
}

func GetResultsDir() string {
	return filepath.Join(GetBaseDir(), "result")
}
func GetLogsDir() string {
	return filepath.Join(GetBaseDir(), "log")
}

func GetFileName() (string, string) {
	objectKey := serviceConfig.ApplicationSetting.ObjectKey
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return fileNameWithoutExt, fileName

}
