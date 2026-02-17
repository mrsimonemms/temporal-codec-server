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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

type MongoDBConfig struct {
	DB         string
	Collection string
	URI        string
}

type MongoDB struct {
	cfg        *MongoDBConfig
	client     *mongo.Client
	collection *mongo.Collection
	ctx        context.Context
	timeout    time.Duration
}

type MongoDBRecord struct {
	Key       string    `bson:"_id,omitempty"`
	Value     string    `bson:"value,omitempty"`
	CreatedAt time.Time `bson:"created_at,omitempty"`
}

func (m *MongoDB) Close() error {
	return m.client.Disconnect(m.ctx)
}

func (m *MongoDB) Get(key uuid.UUID) (value []byte, err error) {
	ctx, cancel := context.WithTimeout(m.ctx, m.timeout)
	defer cancel()

	var result MongoDBRecord
	if err := m.collection.FindOne(ctx, MongoDBRecord{
		Key: key.String(),
	}).Decode(&result); err != nil {
		return nil, fmt.Errorf("error getting mongodb record: %w", err)
	}

	return []byte(result.Value), nil
}

func (m *MongoDB) GetTypeID() string {
	return "mongodb"
}

func (m *MongoDB) Save(key uuid.UUID, value []byte) error {
	ctx, cancel := context.WithTimeout(m.ctx, m.timeout)
	defer cancel()

	if _, err := m.collection.InsertOne(ctx, MongoDBRecord{
		Key:       key.String(),
		Value:     string(value),
		CreatedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("error saving mongodb record: %w", err)
	}

	return nil
}

func NewMongoDB(ctx context.Context, opts *MongoDBConfig, expiration ...time.Duration) (*MongoDB, error) {
	if opts == nil {
		opts = &MongoDBConfig{}
	}
	if opts.URI == "" {
		opts.URI = "mongodb://localhost:27017"
	}

	if opts.Collection == "" {
		opts.Collection = "temporal"
	}

	if len(expiration) == 0 {
		// Expration not set - don't expire
		expiration = []time.Duration{0}
	}

	if opts.DB == "" {
		// No DB specified - ensure in opts
		cs, err := connstring.Parse(opts.URI)
		if err != nil {
			return nil, fmt.Errorf("error parsing connection string: %w", err)
		}

		if cs.Database == "" {
			return nil, fmt.Errorf("database must be specified in mongodb uri")
		}

		opts.DB = cs.Database
	}

	timeout := 10 * time.Second

	lctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(opts.URI)
	client, err := mongo.Connect(lctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection to mongodb: %w", err)
	}
	if err := client.Ping(lctx, nil); err != nil {
		return nil, fmt.Errorf("unable to connect to mongodb: %w", err)
	}

	database := client.Database(opts.DB)
	collection := database.Collection(opts.Collection)

	indexName := "created_at_ttl"
	if exp := expiration[0]; exp > 0 {
		expiresIndex := mongo.IndexModel{
			Keys: bson.M{"created_at": 1},
			Options: options.Index().
				SetExpireAfterSeconds(int32(exp.Seconds())).
				SetName(indexName),
		}

		if _, err := collection.Indexes().CreateOne(lctx, expiresIndex); err != nil {
			return nil, fmt.Errorf("error creating expires index: %w", err)
		}
	} else {
		if _, err := collection.Indexes().DropOne(context.Background(), indexName); err != nil {
			return nil, fmt.Errorf("error deleting expires index: %w", err)
		}
	}

	return &MongoDB{
		collection: collection,
		cfg:        opts,
		client:     client,
		ctx:        ctx,
		timeout:    timeout,
	}, nil
}
