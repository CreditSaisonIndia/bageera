package fileUtilityWrapper

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"
)

var LOGGER *zap.SugaredLogger

func CreateReader(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func DeleteDirIfExist(outputChunkDir string) {
	LOGGER := customLogger.GetLogger()
	if _, err := os.Stat(outputChunkDir); os.IsNotExist(err) {
		LOGGER.Info("Directory does not exist:", outputChunkDir)
	} else {
		// Directory exists, so delete it
		err := os.RemoveAll(outputChunkDir)
		if err != nil {
			LOGGER.Info("Error deleting directory:", err)
		} else {
			LOGGER.Info("Directory deleted successfully:", outputChunkDir)
		}
	}
}

type progressWriter struct {
	total  int64
	length int64
	prefix string
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	pw.length += int64(n)
	progress := float64(pw.length) / float64(pw.total) * 100
	fmt.Printf("\r%s%.2f%%", pw.prefix, progress)
	return n, nil
}

func S3FileDownload() (path string) {
	LOGGER = customLogger.GetLogger()
	//region := config.Get("region")
	bucketName := serviceConfig.Get("bucketName")
	objectKey := serviceConfig.Get("objectKey")
	// Specify the S3 endpoint for your region

	// Create a new AWS session with the S3 endpoint
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_DEFAULT_REGION")),
	})

	if err != nil {
		LOGGER.Error("Error creating AWS session:", err)
	}

	// Create an S3 service client
	s3Client := s3.New(sess)

	// Download the file from S3
	downloadOutput, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		LOGGER.Fatal("Error downloading file from S3:", err)
	}

	defer func(file *s3.GetObjectOutput) {
		err := file.Body.Close()
		if err != nil {
			LOGGER.Fatal(err)
		}
	}(downloadOutput)

	//Provide a local file path for saving the downloaded file
	baseDir := utils.GetBaseDir()
	_, fileName := utils.GetFileName()
	downloadPath := filepath.Join(baseDir, fileName)
	localFile, err := os.Create(downloadPath)
	// Get the absolute path
	absolutePath, err := filepath.Abs(downloadPath)

	// Check for errors
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Print the absolute path
	LOGGER.Info("Absolute path of the file:", absolutePath)

	if err != nil {
		LOGGER.Fatal("Error creating local file:", err)
	}
	defer func(localFile *os.File) {
		err := localFile.Close()
		if err != nil {
			LOGGER.Fatal("Error creating local file:", err)
		}
	}(localFile)

	if serviceConfig.ApplicationSetting.RunType == "local" {

		contentLength := *downloadOutput.ContentLength

		// Create a progress writer for reporting
		progressWriter := &progressWriter{total: contentLength, prefix: fmt.Sprintf("Downloading %s: ", downloadPath)}

		// Copy the S3 file content to the local file
		_, err = io.Copy(localFile, io.TeeReader(downloadOutput.Body, progressWriter))
		if err != nil {
			LOGGER.Fatal("Error copying file content:", err)
		}
	} else {
		// Copy the S3 file content to the local file
		LOGGER.Info("Downloading File...")
		_, err = io.Copy(localFile, downloadOutput.Body)
		if err != nil {
			LOGGER.Fatal("Error copying file content:", err)
		}
	}

	LOGGER.Info("\nFile downloaded successfully to %s\n", downloadPath)
	return downloadPath
}

//func AddLogFileSugar() (*os.File, error) {
//	LOGGER = customLogger.GetLogger()
//	logDirectory := utils.GetLogsDir()
//
//	logFilePath := filepath.Join(logDirectory, "log.txt")
//	logFile, err := os.Create(logFilePath)
//	if err != nil {
//		LOGGER.Info("Error creating log file:", err)
//		return nil, err
//	}
//
//	// Create a zapcore.EncoderConfig and specify the desired output format
//	encoderConfig := zap.NewProductionEncoderConfig()
//	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
//
//	// Create a zapcore.WriteSyncer for the console and the log file
//	consoleDebugging := zapcore.Lock(os.Stdout)
//	fileWriting := zapcore.AddSync(logFile)
//
//	// Create a MultiWriteSyncer to write logs to both console and file
//	multiWriteSyncer := zapcore.NewMultiWriteSyncer(consoleDebugging, fileWriting)
//
//	// Configure the Zap logger with the desired log level, encoder, and output
//	core := zapcore.NewCore(
//		zapcore.NewJSONEncoder(encoderConfig),
//		multiWriteSyncer,
//		zap.NewAtomicLevel(),
//	)
//
//	// Create the logger
//	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
//
//	// Create a SugaredLogger for convenient logging
//	LOGGER = logger.Sugar()
//
//	return logFile, nil
//}

func AddLogFile() (*os.File, error) {
	logDirectory := utils.GetLogsDir()

	// Create the directory if it doesn't exist
	err := os.MkdirAll(logDirectory, os.ModePerm)
	if err != nil {
		LOGGER.Info("Error creating log directory:", err)
		return nil, err
	}

	// Create or open a log file inside the directory
	logFilePath := filepath.Join(logDirectory, "log.txt")
	logFile, err := os.Create(logFilePath)
	if err != nil {
		LOGGER.Info("Error creating log file:", err)
		return nil, err
	}

	// Set the log output to both console and the log file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	return logFile, nil
}

func Copy(fileName, fileNameWithoutExt, sourcePath, destinationPath string) {

	err := copyFile(sourcePath, filepath.Join(destinationPath, fileNameWithoutExt+"_1.csv"))
	if err != nil {
		LOGGER.Info("Error copying the file:", err)
		return
	}

}

func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}
