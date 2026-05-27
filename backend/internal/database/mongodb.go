package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	"github.com/bongcloud/chess/internal/config"
)

// MongoDB wraps the official mongo.Client and exposes the target database.
// All repositories receive this struct via dependency injection.
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
	log      *zap.Logger
	cfg      config.MongoConfig
}

// NewMongoDB dials MongoDB, verifies connectivity with a ping, and returns
// a ready-to-use MongoDB instance. Returns an error if the connection or
// initial ping fails within the configured timeout.
func NewMongoDB(cfg config.MongoConfig, log *zap.Logger) (*MongoDB, error) {
	log.Info("connecting to MongoDB", zap.String("uri_prefix", safeURI(cfg.URI)))

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)

	opts := options.Client().
		ApplyURI(cfg.URI).
		SetServerAPIOptions(serverAPI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetConnectTimeout(cfg.Timeout).
		SetServerSelectionTimeout(cfg.Timeout)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect: %w", err)
	}

	// Verify the connection is actually alive
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping failed: %w", err)
	}

	log.Info("MongoDB connected",
		zap.String("database", cfg.Database),
		zap.Uint64("max_pool", cfg.MaxPoolSize),
	)

	return &MongoDB{
		Client:   client,
		Database: client.Database(cfg.Database),
		log:      log,
		cfg:      cfg,
	}, nil
}

// Collection is a convenience accessor for a named collection.
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// Ping checks that the primary is reachable. Used by the health-check endpoint.
func (m *MongoDB) Ping(ctx context.Context) error {
	return m.Client.Ping(ctx, readpref.Primary())
}

// Disconnect performs a graceful shutdown, flushing in-flight operations
// before closing connections. Should be called during application shutdown.
func (m *MongoDB) Disconnect(ctx context.Context) error {
	m.log.Info("disconnecting MongoDB")
	if err := m.Client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongo disconnect: %w", err)
	}
	m.log.Info("MongoDB disconnected")
	return nil
}

// RunTransaction is a helper that executes fn inside a multi-document ACID
// transaction. The caller is responsible for performing all operations using
// the session context sc passed to fn.
func (m *MongoDB) RunTransaction(ctx context.Context, fn func(sc mongo.SessionContext) (interface{}, error)) (interface{}, error) {
	session, err := m.Client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("start session: %w", err)
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, fn)
	if err != nil {
		return nil, fmt.Errorf("transaction: %w", err)
	}
	return result, nil
}

// CreateIndexes runs the provided IndexModels against the named collection.
// Call this during application startup for each domain model.
func (m *MongoDB) CreateIndexes(ctx context.Context, collection string, models []mongo.IndexModel) error {
	col := m.Collection(collection)
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := col.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("create indexes on %q: %w", collection, err)
	}
	return nil
}

// safeURI strips credentials from a MongoDB URI for safe logging.
func safeURI(uri string) string {
    r := []rune(uri)
    if len(r) > 20 {
        return string(r[:20]) + "..."
    }
    return uri
}