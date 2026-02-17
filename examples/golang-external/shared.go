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

package golang

import (
	"context"
	"fmt"
	"os"

	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/external"
	"github.com/redis/go-redis/v9"
)

func LoadConnection(driver string) (external.Connection, error) {
	ctx := context.Background()
	switch driver {
	case "mongodb":
		return external.NewMongoDB(ctx, &external.MongoDBConfig{
			DB:         os.Getenv("MONGODB_DB"),
			Collection: os.Getenv("MONGODB_COLLECTION"),
			URI:        os.Getenv("MONGODB_URL"),
		})
	case "redis":
		return external.NewRedis(ctx, &redis.Options{
			Addr: os.Getenv("REDIS_ADDRESS"),
		})
	case "s3":
		return external.NewS3(ctx, &external.S3Config{
			Region:          os.Getenv("S3_REGION"),
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			BucketName:      os.Getenv("S3_BUCKETNAME"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			UsePathStyle:    os.Getenv("S3_USE_PATH_STYLE") == "true",
		})
	}
	return nil, fmt.Errorf("unable to create connection for driver: %s", driver)
}
