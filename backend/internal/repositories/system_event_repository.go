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

type SystemEventRepository interface {
	Create(ctx context.Context, systemEvent *models.SystemEvent) error
	Get(ctx context.Context, id string) (*models.SystemEvent, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, systemEvent *models.SystemEvent) error
	List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error)
	ListByByService(ctx context.Context, service string, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error)
	ListByByType(ctx context.Context, eventType string, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error)
}

type DynamoSystemEventRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoSystemEventRepository(client *dynamodb.Client, tableName string) *DynamoSystemEventRepository {
	return &DynamoSystemEventRepository{client: client, tableName: tableName}
}

func (r *DynamoSystemEventRepository) Create(ctx context.Context, systemEvent *models.SystemEvent) error {
	now := time.Now().UTC()
	systemEvent.CreatedAt = now
	systemEvent.UpdatedAt = now
	item, err := attributevalue.MarshalMap(systemEvent)
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

func (r *DynamoSystemEventRepository) Get(ctx context.Context, id string) (*models.SystemEvent, error) {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.tableName), Key: key})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	var systemEvent models.SystemEvent
	if err := attributevalue.UnmarshalMap(result.Item, &systemEvent); err != nil {
		return nil, err
	}
	return &systemEvent, nil
}

func (r *DynamoSystemEventRepository) Update(ctx context.Context, systemEvent *models.SystemEvent) error {
	systemEvent.UpdatedAt = time.Now().UTC()
	item, err := attributevalue.MarshalMap(systemEvent)
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

func (r *DynamoSystemEventRepository) Delete(ctx context.Context, id string) error {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.tableName), Key: key})
	return err
}

func (r *DynamoSystemEventRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error) {
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
	var systemEvents []*models.SystemEvent
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &systemEvents); err != nil {
		return nil, nil, err
	}
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return systemEvents, nextKey, nil
}

func (r *DynamoSystemEventRepository) ListByByService(ctx context.Context, service string, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("ByService"),
		KeyConditionExpression: aws.String("service = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: service},
		},
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var systemEvents []*models.SystemEvent
	_ = attributevalue.UnmarshalListOfMaps(result.Items, &systemEvents)
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return systemEvents, nextKey, nil
}

func (r *DynamoSystemEventRepository) ListByByType(ctx context.Context, eventType string, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("ByType"),
		KeyConditionExpression: aws.String("#t = :pk"),
		ExpressionAttributeNames: map[string]string{
			"#t": "type",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: eventType},
		},
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var systemEvents []*models.SystemEvent
	_ = attributevalue.UnmarshalListOfMaps(result.Items, &systemEvents)
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return systemEvents, nextKey, nil
}
