package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func GetMetadataUsecaseDir(usecaseDir string) string {
	fmt.Printf("Fetching EFS base path value From Settings : %s\n", serviceConfig.ApplicationSetting.EfsBasePath)
	fmt.Printf("Fetching EFS base path value From env : %s\n", os.Getenv("efsBasePath"))
	efsBasePath := os.Getenv("efsBasePath")
	objectKey := GetMetadataObjectKey()
	dirPath := filepath.Dir(objectKey)
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return filepath.Join(efsBasePath, dirPath, fileNameWithoutExt, usecaseDir)

}

func GetMetadataUsecasesDir(usecaseDir ...string) string {
	fmt.Printf("Fetching EFS base path value From Settings : %s\n", serviceConfig.ApplicationSetting.EfsBasePath)
	fmt.Printf("Fetching EFS base path value From env : %s\n", os.Getenv("efsBasePath"))
	efsBasePath := os.Getenv("efsBasePath")
	objectKey := GetMetadataObjectKey()
	dirPath := filepath.Dir(objectKey)
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return filepath.Join(efsBasePath, dirPath, fileNameWithoutExt, filepath.Join(usecaseDir...))

}

func GetMetadataBaseDir() string {
	fmt.Printf("Fetching EFS base path value From Settings : %s\n", serviceConfig.ApplicationSetting.EfsBasePath)
	fmt.Printf("Fetching EFS base path value From env : %s\n", os.Getenv("efsBasePath"))
	efsBasePath := os.Getenv("efsBasePath")
	objectKey := GetMetadataObjectKey()
	dirPath := filepath.Dir(objectKey)
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return filepath.Join(efsBasePath, dirPath, fileNameWithoutExt)

}

func GetInvalidBaseDir() string {
	return GetMetadataUsecaseDir("invalid")
}

func GetRelativeInvalidBaseDir() string {
	return GetRelativeMetadataUsecaseDir("invalid")
}

func GetRelativeMetadataUsecaseDir(usecaseDir string) string {
	objectKey := GetMetadataObjectKey()
	dirPath := filepath.Dir(objectKey)
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return filepath.Join(dirPath, fileNameWithoutExt, usecaseDir)

}

func GetMetadataObjectKey() string {
	objectKey := serviceConfig.ApplicationSetting.ObjectKey

	parts1 := strings.Split(objectKey, "/")

	parts1[0] = "metadata"

	// Join the parts back into a string
	metadataObjectKey := strings.Join(parts1, "/")

	dirPath := filepath.Dir(metadataObjectKey)
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)

	return filepath.Join(dirPath, fileName)

}

func GetJobTypeFromPath() string {
	path := serviceConfig.ApplicationSetting.ObjectKey
	parts := strings.Split(path, "/")

	// Iterate through the parts to find "insert"
	var extractedPart string
	for _, part := range parts {
		if part == "insert" || part == "delete" || part == "update" {
			extractedPart = part
			break
		}
	}
	return extractedPart

}

func GetInvalidObjectKey() string {
	objectKey := serviceConfig.ApplicationSetting.ObjectKey

	parts1 := strings.Split(objectKey, "/")

	parts1[0] = "metadata"

	// Join the parts back into a string
	invalidObjectKey := strings.Join(parts1, "/")

	dirPath := filepath.Dir(invalidObjectKey)
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)

	return filepath.Join(dirPath, fileName)

}

func GetChunksDir() string {
	return GetMetadataUsecaseDir("chunks")
}

func GetExistenceChunksDir() string {
	return GetMetadataUsecasesDir("chunks")
}

func GetResultsDir() string {
	return GetMetadataUsecaseDir("result")
}
func GetLogsDir() string {
	return GetMetadataUsecaseDir("log")
}

func GetFileName() (string, string) {
	objectKey := serviceConfig.ApplicationSetting.ObjectKey
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return fileNameWithoutExt, fileName

}

func GetFileNameFromPath(path string) (string, string) {
	objectKey := serviceConfig.ApplicationSetting.ObjectKey
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return fileNameWithoutExt, fileName

}

func PrettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
