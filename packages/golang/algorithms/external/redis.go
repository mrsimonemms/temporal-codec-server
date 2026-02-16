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
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	ctx        context.Context
	client     *redis.Client
	expiration time.Duration
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) GetTypeID() string {
	return "redis"
}

func (r *Redis) Get(key uuid.UUID) (value []byte, err error) {
	val, err := r.client.Get(r.ctx, key.String()).Result()
	if err != nil {
		return nil, err
	}

	return []byte(val), nil
}

func (r *Redis) Save(key uuid.UUID, value []byte) error {
	return r.client.Set(r.ctx, key.String(), value, r.expiration).Err()
}

func NewRedis(ctx context.Context, opts *redis.Options, expiration ...time.Duration) (*Redis, error) {
	client := redis.NewClient(opts)

	if len(expiration) == 0 {
		// Expration not set - don't expire
		expiration = []time.Duration{0}
	}

	// Test the connection
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("error connecting to redis: %w", err)
	}

	return &Redis{
		ctx:        ctx,
		client:     client,
		expiration: expiration[0],
	}, nil
}
