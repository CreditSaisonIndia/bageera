package multipartUpload

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/sync/errgroup"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const multipartUploadMinPartSize = 5 * 1024 * 1024

// MultipartUploadConfig defines multipart upload specifications.
// Example:
//
//	aws.MultipartUploadConfig{
//		Key:      "some-unique-key",
//		Filename: "some-filename",
//		Mime:     "some-valid-mime-type",
//	}
//
//	aws.MultipartUploadConfig{
//		Key:      "some-unique-key",
//		Filename: "some-filename",
//		Mime:     "some-valid-mime-type",
//		Bucket:   "some-bucket",
//		Workers:  2,
//		Expiry:   86400, // 1 day
//		Retry:    3,
//		Size:     10 * 1024 * 1024, // 10MB
//	}
type MultipartUploadConfig struct {
	// Object key for which the multipart upload was initiated. Required.
	Key string

	// Used to name file for downloading. Required.
	Filename string

	// The MIME type representing the format of the object data. Required.
	Mime string

	// The name of the bucket where the object is stored. If not set, it falls
	// back to S3 level root bucket.
	Bucket string

	// Specifis the maximum amount of goroutines to run at a time to handle part
	// uploads. If not set, it falls back to 1. The higher it goes, the more
	// stress on memory. An ideal value would be no more than 2 for better
	// memory efficiency.
	Workers int

	// Specifies the maximum time (in seconds) a failed/dangling multipart
	// upload can live in a S3 bucket before being automatically removed using
	// lifecycle management service even if you forget to call abort. Parts of
	// an incomplete multipart upload are invisible however incur fees for the
	// allocations. If not set, it falls back to 2 days.
	Expiry int

	// Forces retrying a failed part upload in case of an error. If not set, no
	// retry.
	Retry int

	// Specifies the minimum buffer size in bytes for each upload part. It is
	// defaulted to 5 MB but the last part has no limit.
	// Ref: https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html
	Size int
}

type MultipartUpload struct {
	client           *s3.Client
	config           MultipartUploadConfig
	goroutineGroup   *errgroup.Group
	goroutineContext context.Context
	buffer           *bytes.Buffer
	workers          chan struct{}
	mux              *sync.Mutex
	parts            map[int]*string
	cursor           int
	id               string
	abort            bool
}

// Write writes a new data to a buffer for the part upload.
func (m *MultipartUpload) Write(row string) error {
	LOGGER := customLogger.GetLogger()
	if _, err := m.buffer.WriteString(row); err != nil {
		LOGGER.Error("Error while string to buffer")
		m.mux.Lock()
		m.abort = true
		m.mux.Unlock()

		return fmt.Errorf("write string: %w", err)
	}

	if m.buffer.Len() < m.config.Size {
		return nil
	}
	// m.workers = make(chan struct{}, 10)

	m.workers <- struct{}{}

	buffer := &bytes.Buffer{}
	buffer.Write(m.buffer.Bytes())
	part := m.cursor

	m.goroutineGroup.Go(func() error {
		err := m.upload(buffer, part)
		if err != nil {
			LOGGER.Error("Error while upload")
			m.mux.Lock()
			m.abort = true
			m.mux.Unlock()
		}

		<-m.workers

		return err
	})

	m.buffer.Reset()
	m.cursor++

	return nil
}

// Flush uploads last remaining part, waits for all ongoing uploads to end
// before completing the whole upload operation. In case an error occurs,
// `abort` method is called.
func (m *MultipartUpload) Flush(ctx context.Context) (int, error) {
	m.goroutineGroup.Go(func() error {
		return m.upload(m.buffer, m.cursor)
	})

	if err := m.goroutineGroup.Wait(); err != nil {
		m.mux.Lock()
		m.abort = true
		m.mux.Unlock()

		return 0, fmt.Errorf("goroutine group wait: %w", err)
	}

	total := len(m.parts)
	parts := make([]types.CompletedPart, total)

	for i, tag := range m.parts {
		int32 := int32(i)
		parts[i-1] = types.CompletedPart{
			PartNumber: &int32,
			ETag:       tag,
		}
	}

	inp := &s3.CompleteMultipartUploadInput{
		Bucket:          &m.config.Bucket,
		Key:             &m.config.Key,
		UploadId:        &m.id,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: parts},
	}

	if _, err := m.client.CompleteMultipartUpload(ctx, inp); err != nil {
		m.mux.Lock()
		m.abort = true
		m.mux.Unlock()

		return 0, fmt.Errorf("complete multipart upload: %w", err)
	}

	return total, nil
}

// upload tries to upload a new part and saves its ETag. In case all tries are
// exhausted, last error is returned.
func (m *MultipartUpload) upload(buffer *bytes.Buffer, part int) error {
	LOGGER := customLogger.GetLogger()
	if buffer.Len() == 0 {
		return nil
	}

	var (
		try int
		res *s3.UploadPartOutput
		err error
	)
	partInt32 := int32(part)
	inp := &s3.UploadPartInput{
		Bucket:     &m.config.Bucket,
		Key:        &m.config.Key,
		UploadId:   &m.id,
		PartNumber: &partInt32,
		Body:       buffer,
	}

	opt := s3.WithAPIOptions(
		v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware,
	)

	for try <= m.config.Retry {
		select {
		case <-m.goroutineContext.Done():
			return m.goroutineContext.Err()
		default:
		}

		res, err = m.client.UploadPart(m.goroutineContext, inp, opt)
		if err == nil {
			LOGGER.Info("Part Done --- > ", partInt32)
			m.mux.Lock()
			m.parts[part] = res.ETag
			m.mux.Unlock()

			return nil
		}

		if buffer.Len() == 0 {
			LOGGER.Error("Error buffer len is Zero")
			return fmt.Errorf("upload part: exiting retry due to an empty buffer: %w", err)
		}

		try++
	}

	return fmt.Errorf("Error upload part: %w", err)
}

// Abort is automatically called after whole upload operation to determine if
// the operation should be aborted or not.
func (m *MultipartUpload) Abort() error {
	LOGGER := customLogger.GetLogger()
	if !m.abort {
		return nil
	}

	inp := &s3.AbortMultipartUploadInput{
		Bucket:   &m.config.Bucket,
		Key:      &m.config.Key,
		UploadId: &m.id,
	}

	if _, err := m.client.AbortMultipartUpload(context.Background(), inp); err != nil {
		LOGGER.Error("Error while m.client.AbortMultipartUpload")
		return fmt.Errorf("abort multipart upload: %w", err)
	}

	return nil
}
