package multipartUpload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/sync/errgroup"
)

type S3 struct {
	client *s3.Client
	signer *s3.PresignClient
	bucket string
}

func NewS3(config aws.Config, bucket string) S3 {
	client := s3.NewFromConfig(config)

	return S3{
		client: client,
		signer: s3.NewPresignClient(client),
		bucket: bucket,
	}
}

func (s S3) CreateMultipartUpload(ctx context.Context, cfg MultipartUploadConfig) (*MultipartUpload, error) {
	if cfg.Key == "" || cfg.Filename == "" {
		return nil, errors.New("required field: Key")
	}

	if cfg.Filename == "" {
		return nil, errors.New("required field: Filename")
	}

	if cfg.Mime == "" {
		return nil, errors.New("required field: Mime")
	}

	// if cfg.Size != 0 && cfg.Size < multipartUploadMinPartSize {
	// 	return nil, fmt.Errorf("invalid value: Size: minimum required value is %d", multipartUploadMinPartSize)
	// }

	if cfg.Bucket == "" {
		cfg.Bucket = s.bucket
	}

	if cfg.Workers == 0 {
		cfg.Workers = 5
	}

	if cfg.Expiry == 0 {
		cfg.Expiry = 172800
	}
	exp := time.Now().Add(time.Second * time.Duration(cfg.Expiry))

	if cfg.Size == 0 {
		cfg.Size = multipartUploadMinPartSize
	}

	inp := &s3.CreateMultipartUploadInput{
		Bucket:             &cfg.Bucket,
		Key:                &cfg.Key,
		ContentType:        &cfg.Mime,
		Expires:            &exp,
		ContentDisposition: aws.String(fmt.Sprintf(`attachment; filename="%s"`, cfg.Filename)),
	}

	res, err := s.client.CreateMultipartUpload(ctx, inp)
	if err != nil {
		return nil, fmt.Errorf("create multipart upload: %w", err)
	}

	erg, erc := errgroup.WithContext(ctx)

	return &MultipartUpload{
		client:           s.client,
		config:           cfg,
		goroutineGroup:   erg,
		goroutineContext: erc,
		buffer:           &bytes.Buffer{},
		workers:          make(chan struct{}, cfg.Workers),
		mux:              &sync.Mutex{},
		parts:            make(map[int]*string),
		cursor:           1,
		id:               *res.UploadId,
	}, nil
}
