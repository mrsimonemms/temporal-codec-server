/*
 * Copyright 2025 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package external

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

type S3Config struct {
	Region          string
	Endpoint        string // Leave empty for AWS, set for non-AWS
	BucketName      string // Add bucket name here
	AccessKeyID     string // Optional for AWS (can use default credential chain)
	SecretAccessKey string // Optional for AWS
	UsePathStyle    bool   // Set true for non-AWS, false for AWS
}

type S3 struct {
	bucketName string
	client     *s3.Client
	ctx        context.Context
}

func (s *S3) Close() error {
	return nil
}

func (s *S3) Get(key uuid.UUID) (value []byte, err error) {
	result, err := s.client.GetObject(s.ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key.String()),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting object: %w", err)
	}
	defer func() {
		err = result.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading data: %w", err)
	}

	return bodyBytes, err
}

func (s *S3) GetTypeID() string {
	return "s3"
}

func (s *S3) Save(key uuid.UUID, value []byte) error {
	_, err := s.client.PutObject(s.ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key.String()),
		Body:   strings.NewReader(string(value)),
	})
	return err
}

func NewS3(ctx context.Context, s3cfg *S3Config, expiration ...time.Duration) (*S3, error) {
	var cfg aws.Config
	var err error

	if len(expiration) == 0 {
		// Expration not set - don't expire
		expiration = []time.Duration{0}
	}

	if s3cfg.AccessKeyID != "" && s3cfg.SecretAccessKey != "" {
		// Use the credentials provided
		cfg, err = config.LoadDefaultConfig(
			ctx,
			config.WithRegion(s3cfg.Region),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					s3cfg.AccessKeyID,
					s3cfg.SecretAccessKey,
					"",
				),
			),
		)
	} else {
		// Use the default chain
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s3cfg.Region),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("error creating s3 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if s3cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(s3cfg.Endpoint)
		}
		o.UsePathStyle = s3cfg.UsePathStyle
	})

	if exp := expiration[0]; exp > 0 {
		_, err := client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
			Bucket: aws.String(s3cfg.BucketName),
			LifecycleConfiguration: &types.BucketLifecycleConfiguration{
				Rules: []types.LifecycleRule{
					{
						ID:     aws.String("auto-expire"),
						Status: types.ExpirationStatusEnabled,
						Expiration: &types.LifecycleExpiration{
							Days: aws.Int32(durationToDays(exp)),
						},
					},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("error setting bucket lifecycle policy: %w", err)
		}
	}

	return &S3{
		bucketName: s3cfg.BucketName,
		client:     client,
		ctx:        ctx,
	}, nil
}

func durationToDays(d time.Duration) int32 {
	days := d.Hours() / 24

	if days < 1 {
		return 1
	}

	return int32(math.Ceil(days))
}
