package awsClient

import (
	"bufio"
	"context"
	"os"
	"path/filepath"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient/multipartUpload"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

func UploadDriver(ctx context.Context, s3 multipartUpload.S3, filePath, baseDir string) error {

	LOGGER := customLogger.GetLogger()
	// Dummy data fixtures

	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		LOGGER.Error("Error while opening invalid file : ", err)
		return err
	}
	// Multipart uploader instance
	up, err := s3.CreateMultipartUpload(ctx, multipartUpload.MultipartUploadConfig{
		Key:      filepath.Join(baseDir, filepath.Base(filePath)),
		Filename: filepath.Base(filePath),
		Mime:     "text/csv",
		Bucket:   serviceConfig.ApplicationSetting.BucketName,
	})
	if err != nil {
		LOGGER.Error("Error while creating multipert client : ", err)
		return err
	}
	defer up.Abort()

	// Read fixtures line by line and upload
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		err := up.Write(scanner.Text() + "\n")
		if err != nil {
			LOGGER.Error("Error while up.Write : ", err)
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		LOGGER.Error("Error while scanner.Err : ", err)
		return err
	}

	// Upload (flush) any remaining parts
	tot, err := up.Flush(ctx)
	if err != nil {
		LOGGER.Error("Error while up.Flush : ", err)
		return err
	}

	LOGGER.Info("uploaded parts:", tot)

	return nil
}
