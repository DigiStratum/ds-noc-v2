package featureflags

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Store manages feature flag storage in DynamoDB
// Uses single-table design with PK: FF#<key>, SK: FLAG
type Store struct {
	client    *dynamodb.Client
	tableName string

	// In-memory cache for performance
	mu    sync.RWMutex
	cache map[string]*FeatureFlag
	ttl   time.Time
}

const (
	// Key prefix for feature flags in DynamoDB
	keyPrefix = "FF#"
	// Sort key for flag records
	sortKeyFlag = "FLAG"
	// Cache TTL
	cacheTTL = 30 * time.Second
)

var (
	globalStore *Store
	storeOnce   sync.Once
)

// GetStore returns the singleton Store instance
func GetStore() *Store {
	storeOnce.Do(func() {
		store, err := newStore()
		if err != nil {
			panic(fmt.Sprintf("failed to initialize feature flag store: %v", err))
		}
		globalStore = store
	})
	return globalStore
}

// newStore creates a new Store instance
func newStore() (*Store, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		tableName = "ds-noc-v2"
	}

	return &Store{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
		cache:     make(map[string]*FeatureFlag),
	}, nil
}

// Get retrieves a feature flag by key
func (s *Store) Get(ctx context.Context, key string) (*FeatureFlag, error) {
	// Check cache first
	s.mu.RLock()
	if flag, ok := s.cache[key]; ok && time.Now().Before(s.ttl) {
		s.mu.RUnlock()
		return flag, nil
	}
	s.mu.RUnlock()

	// Fetch from DynamoDB
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: keyPrefix + key},
			"SK": &types.AttributeValueMemberS{Value: sortKeyFlag},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get feature flag: %w", err)
	}

	if result.Item == nil {
		return nil, nil // Flag not found
	}

	var flag FeatureFlag
	if err := attributevalue.UnmarshalMap(result.Item, &flag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal feature flag: %w", err)
	}

	// Update cache
	s.mu.Lock()
	s.cache[key] = &flag
	s.ttl = time.Now().Add(cacheTTL)
	s.mu.Unlock()

	return &flag, nil
}

// List returns all feature flags
func (s *Store) List(ctx context.Context) ([]*FeatureFlag, error) {
	// Query all flags using begins_with on PK
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("begins_with(PK, :prefix) AND SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: keyPrefix},
			":sk":     &types.AttributeValueMemberS{Value: sortKeyFlag},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list feature flags: %w", err)
	}

	flags := make([]*FeatureFlag, 0, len(result.Items))
	for _, item := range result.Items {
		var flag FeatureFlag
		if err := attributevalue.UnmarshalMap(item, &flag); err != nil {
			return nil, fmt.Errorf("failed to unmarshal feature flag: %w", err)
		}
		flags = append(flags, &flag)
	}

	// Update cache
	s.mu.Lock()
	for _, flag := range flags {
		s.cache[flag.Key] = flag
	}
	s.ttl = time.Now().Add(cacheTTL)
	s.mu.Unlock()

	return flags, nil
}

// Save creates or updates a feature flag
func (s *Store) Save(ctx context.Context, flag *FeatureFlag) error {
	flag.UpdatedAt = time.Now().UTC()

	item, err := attributevalue.MarshalMap(flag)
	if err != nil {
		return fmt.Errorf("failed to marshal feature flag: %w", err)
	}

	// Add primary key
	item["PK"] = &types.AttributeValueMemberS{Value: keyPrefix + flag.Key}
	item["SK"] = &types.AttributeValueMemberS{Value: sortKeyFlag}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save feature flag: %w", err)
	}

	// Update cache
	s.mu.Lock()
	s.cache[flag.Key] = flag
	s.mu.Unlock()

	return nil
}

// Delete removes a feature flag
func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: keyPrefix + key},
			"SK": &types.AttributeValueMemberS{Value: sortKeyFlag},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete feature flag: %w", err)
	}

	// Remove from cache
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()

	return nil
}

// InvalidateCache clears the in-memory cache
func (s *Store) InvalidateCache() {
	s.mu.Lock()
	s.cache = make(map[string]*FeatureFlag)
	s.ttl = time.Time{}
	s.mu.Unlock()
}
