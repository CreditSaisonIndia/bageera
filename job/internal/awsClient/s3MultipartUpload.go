package awsClient

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const PartSize = 10 * 1024 * 1024
const RETRIES = 2

var s3session *s3.S3

type partUploadResult struct {
	completedPart *s3.CompletedPart
	err           error
}

var wg = sync.WaitGroup{}
var ch = make(chan partUploadResult)

func S3MutiPartUpload(invalidGoroutinesWaitGroup *sync.WaitGroup) {
	defer invalidGoroutinesWaitGroup.Done()

	LOGGER := customLogger.GetLogger()

	Init()
	//source efsBasePth+objectKey
	invalidBaseDir := utils.GetInvalidBaseDir()
	fileNameWithoutExt, _ := utils.GetFileName()
	// _, fileName := utils.GetFileNameFromPath(serviceConfig.ApplicationSetting.ObjectKey)

	file, err := os.Open(filepath.Join(invalidBaseDir, fileNameWithoutExt+"_invalid.csv"))
	if err != nil {
		LOGGER.Error("Error while os.Open: ", err)
		return
	}
	defer file.Close()

	// stat, _ := file.Stat()
	// fileSize := stat.Size()

	bufferedReader := bufio.NewReader(file)

	expiryDate := time.Now().AddDate(0, 0, 1)

	createdResp, err := s3session.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket:  aws.String(serviceConfig.ApplicationSetting.BucketName),
		Key:     aws.String(serviceConfig.ApplicationSetting.InvalidObjectKey),
		Expires: &expiryDate,
	})

	if err != nil {
		LOGGER.Error("Error while s3session.CreateMultipartUpload : ", err)
		return
	}

	// var start, currentSize int
	// var remaining = int(fileSize)
	var partNum = 1
	var completedParts []*s3.CompletedPart
	for {
		// Read the next chunk from the file
		data := make([]byte, PartSize)
		n, err := bufferedReader.Read(data)
		if err != nil && err != io.EOF {
			LOGGER.Error("Error while reading file: ", err)
			return
		}

		if n == 0 {
			break
		}

		wg.Add(1)
		go uploadToS3(createdResp, data[:n], partNum, &wg)

		LOGGER.Info("Upload of part ", partNum, " started")
		partNum++
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		if result.err != nil {
			_, err = s3session.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
				Bucket:   aws.String(serviceConfig.ApplicationSetting.BucketName),
				Key:      aws.String(serviceConfig.ApplicationSetting.InvalidObjectKey),
				UploadId: createdResp.UploadId,
			})
			if err != nil {
				LOGGER.Error("Error while s3session.AbortMultipartUpload : ", err)
				return
			}
		}
		LOGGER.Info("Uploading of part ", *result.completedPart.PartNumber, " has been finished")
		completedParts = append(completedParts, result.completedPart)
	}

	// Ordering the array based on the PartNumber as each parts could be uploaded in different order!
	sort.Slice(completedParts, func(i, j int) bool {
		return *completedParts[i].PartNumber < *completedParts[j].PartNumber
	})

	// Signalling AWS S3 that the multiPartUpload is finished
	resp, err := s3session.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   createdResp.Bucket,
		Key:      createdResp.Key,
		UploadId: createdResp.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})

	if err != nil {
		LOGGER.Error("Error while s3session.CompleteMultipartUpload :", err)
		return
	} else {
		LOGGER.Info(resp.String())
	}

}

func Init() {
	s3session = s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String(serviceConfig.ApplicationSetting.Region),
	})))
}

func uploadToS3(resp *s3.CreateMultipartUploadOutput, fileBytes []byte, partNum int, wg *sync.WaitGroup) {
	defer wg.Done()
	var try int
	LOGGER.Info("Uploading ", len(fileBytes))
	time.Sleep(10 * time.Second)
	for try <= RETRIES {
		uploadRes, err := s3session.UploadPart(&s3.UploadPartInput{
			Body:          bytes.NewReader(fileBytes),
			Bucket:        resp.Bucket,
			Key:           resp.Key,
			PartNumber:    aws.Int64(int64(partNum)),
			UploadId:      resp.UploadId,
			ContentLength: aws.Int64(int64(len(fileBytes))),
		})
		if err != nil {
			LOGGER.Error("Error while s3session.UploadPart : ", err)
			if try == RETRIES {
				ch <- partUploadResult{nil, err}
				return
			} else {
				try++
				time.Sleep(time.Duration(time.Second * 15))
			}
		} else {
			ch <- partUploadResult{
				&s3.CompletedPart{
					ETag:       uploadRes.ETag,
					PartNumber: aws.Int64(int64(partNum)),
				}, nil,
			}
			return
		}
	}
	ch <- partUploadResult{}
}
