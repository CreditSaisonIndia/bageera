package utils

import (
	"path/filepath"

	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// NewAWSSession creates a new AWS session.
func NewAWSSession() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		// Add any specific configurations if needed.
	})
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// NewS3ServiceClient creates a new S3 service client using the provided AWS session.
func NewS3ServiceClient(sess *session.Session) *s3.S3 {
	svc := s3.New(sess)
	return svc
}

func GetBaseDir() string {
	efsBasePath := serviceConfig.Get("efsBasePath")
	objectKey := serviceConfig.Get("objectKey")
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
	objectKey := serviceConfig.Get("objectKey")
	// Extract the fileName without extension
	fileName := filepath.Base(objectKey)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	return fileNameWithoutExt, fileName

}
