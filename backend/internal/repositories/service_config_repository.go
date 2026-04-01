package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ServiceConfigRepository interface {
	Create(ctx context.Context, serviceConfig *models.ServiceConfig) error
	Get(ctx context.Context, id string) (*models.ServiceConfig, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, serviceConfig *models.ServiceConfig) error
	List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.ServiceConfig, map[string]interface{}, error)
	ListByByName(ctx context.Context, name string, limit int, lastKey map[string]interface{}) ([]*models.ServiceConfig, map[string]interface{}, error)
}

type DynamoServiceConfigRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoServiceConfigRepository(client *dynamodb.Client, tableName string) *DynamoServiceConfigRepository {
	return &DynamoServiceConfigRepository{client: client, tableName: tableName}
}

func (r *DynamoServiceConfigRepository) Create(ctx context.Context, serviceConfig *models.ServiceConfig) error {
	now := time.Now().UTC()
	serviceConfig.CreatedAt = now
	serviceConfig.UpdatedAt = now
	item, err := attributevalue.MarshalMap(serviceConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	return err
}

func (r *DynamoServiceConfigRepository) Get(ctx context.Context, id string) (*models.ServiceConfig, error) {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.tableName), Key: key})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	var serviceConfig models.ServiceConfig
	if err := attributevalue.UnmarshalMap(result.Item, &serviceConfig); err != nil {
		return nil, err
	}
	return &serviceConfig, nil
}

func (r *DynamoServiceConfigRepository) Update(ctx context.Context, serviceConfig *models.ServiceConfig) error {
	serviceConfig.UpdatedAt = time.Now().UTC()
	item, err := attributevalue.MarshalMap(serviceConfig)
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	return err
}

func (r *DynamoServiceConfigRepository) Delete(ctx context.Context, id string) error {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.tableName), Key: key})
	return err
}

func (r *DynamoServiceConfigRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.ServiceConfig, map[string]interface{}, error) {
	input := &dynamodb.ScanInput{TableName: aws.String(r.tableName)}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	if lastKey != nil {
		startKey, _ := attributevalue.MarshalMap(lastKey)
		input.ExclusiveStartKey = startKey
	}
	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var serviceConfigs []*models.ServiceConfig
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &serviceConfigs); err != nil {
		return nil, nil, err
	}
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return serviceConfigs, nextKey, nil
}

func (r *DynamoServiceConfigRepository) ListByByName(ctx context.Context, name string, limit int, lastKey map[string]interface{}) ([]*models.ServiceConfig, map[string]interface{}, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("ByName"),
		KeyConditionExpression: aws.String("#n = :pk"),
		ExpressionAttributeNames: map[string]string{
			"#n": "name",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: name},
		},
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var serviceConfigs []*models.ServiceConfig
	_ = attributevalue.UnmarshalListOfMaps(result.Items, &serviceConfigs)
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return serviceConfigs, nextKey, nil
}
