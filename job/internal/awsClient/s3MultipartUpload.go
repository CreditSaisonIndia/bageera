package awsClient

import (
	"bytes"
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

// var (
// 	BucketName = serviceConfig.ApplicationSetting.BucketName
// 	REGION     = serviceConfig.ApplicationSetting.BucketName
// 	FILE       = "/200MB.zip"
// 	PartSize   = 50_000_000
// 	RETRIES    = 2
// )

const PartSize = 50_000_000
const RETRIES = 2

var s3session *s3.S3

type partUploadResult struct {
	completedPart *s3.CompletedPart
	err           error
}

var wg = sync.WaitGroup{}
var ch = make(chan partUploadResult)

func S3MutiPartUpload() {
	LOGGER := customLogger.GetLogger()

	Init()
	//source efsBasePth+objectKey
	invalidBaseDir := utils.GetInvalidBaseDir()
	_, fileName := utils.GetFileNameFromPath(serviceConfig.ApplicationSetting.ObjectKey)

	file, err := os.Open(filepath.Join(invalidBaseDir, fileName))
	if err != nil {
		LOGGER.Error("Error while os.Open: ", err)
		return
	}
	defer file.Close()

	stat, _ := file.Stat()
	fileSize := stat.Size()

	buffer := make([]byte, fileSize)

	_, _ = file.Read(buffer)

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

	var start, currentSize int
	var remaining = int(fileSize)
	var partNum = 1
	var completedParts []*s3.CompletedPart
	for start = 0; remaining > 0; start += PartSize {
		wg.Add(1)
		if remaining < PartSize {
			currentSize = remaining
		} else {
			currentSize = PartSize
		}
		go uploadToS3(createdResp, buffer[start:start+currentSize], partNum, &wg)

		remaining -= currentSize
		LOGGER.Info("Uplaodind of part ", partNum, " started and remaning is ", remaining)
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
