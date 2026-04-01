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

type MaintenanceWindowRepository interface {
	Create(ctx context.Context, maintenanceWindow *models.MaintenanceWindow) error
	Get(ctx context.Context, id string) (*models.MaintenanceWindow, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, maintenanceWindow *models.MaintenanceWindow) error
	List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error)
	ListByByService(ctx context.Context, service string, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error)
	ListByByStartTime(ctx context.Context, startTime string, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error)
}

type DynamoMaintenanceWindowRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoMaintenanceWindowRepository(client *dynamodb.Client, tableName string) *DynamoMaintenanceWindowRepository {
	return &DynamoMaintenanceWindowRepository{client: client, tableName: tableName}
}

func (r *DynamoMaintenanceWindowRepository) Create(ctx context.Context, maintenanceWindow *models.MaintenanceWindow) error {
	now := time.Now().UTC()
	maintenanceWindow.CreatedAt = now
	maintenanceWindow.UpdatedAt = now
	item, err := attributevalue.MarshalMap(maintenanceWindow)
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

func (r *DynamoMaintenanceWindowRepository) Get(ctx context.Context, id string) (*models.MaintenanceWindow, error) {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.tableName), Key: key})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	var maintenanceWindow models.MaintenanceWindow
	if err := attributevalue.UnmarshalMap(result.Item, &maintenanceWindow); err != nil {
		return nil, err
	}
	return &maintenanceWindow, nil
}

func (r *DynamoMaintenanceWindowRepository) Update(ctx context.Context, maintenanceWindow *models.MaintenanceWindow) error {
	maintenanceWindow.UpdatedAt = time.Now().UTC()
	item, err := attributevalue.MarshalMap(maintenanceWindow)
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

func (r *DynamoMaintenanceWindowRepository) Delete(ctx context.Context, id string) error {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.tableName), Key: key})
	return err
}

func (r *DynamoMaintenanceWindowRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error) {
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
	var maintenanceWindows []*models.MaintenanceWindow
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &maintenanceWindows); err != nil {
		return nil, nil, err
	}
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return maintenanceWindows, nextKey, nil
}

func (r *DynamoMaintenanceWindowRepository) ListByByService(ctx context.Context, service string, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error) {
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
	var maintenanceWindows []*models.MaintenanceWindow
	_ = attributevalue.UnmarshalListOfMaps(result.Items, &maintenanceWindows)
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return maintenanceWindows, nextKey, nil
}

func (r *DynamoMaintenanceWindowRepository) ListByByStartTime(ctx context.Context, startTime string, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("ByStartTime"),
		KeyConditionExpression: aws.String("start_time = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: startTime},
		},
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var maintenanceWindows []*models.MaintenanceWindow
	_ = attributevalue.UnmarshalListOfMaps(result.Items, &maintenanceWindows)
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return maintenanceWindows, nextKey, nil
}
