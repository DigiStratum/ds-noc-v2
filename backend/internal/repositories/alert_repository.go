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

type AlertRepository interface {
	Create(ctx context.Context, alert *models.Alert) error
	Get(ctx context.Context, id string) (*models.Alert, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, alert *models.Alert) error
	List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error)
	ListByByService(ctx context.Context, serviceID string, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error)
	ListByByTimestamp(ctx context.Context, timestamp string, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error)
}

type DynamoAlertRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoAlertRepository(client *dynamodb.Client, tableName string) *DynamoAlertRepository {
	return &DynamoAlertRepository{client: client, tableName: tableName}
}

func (r *DynamoAlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	now := time.Now().UTC()
	alert.CreatedAt = now
	alert.UpdatedAt = now
	item, err := attributevalue.MarshalMap(alert)
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

func (r *DynamoAlertRepository) Get(ctx context.Context, id string) (*models.Alert, error) {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.tableName), Key: key})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	var alert models.Alert
	if err := attributevalue.UnmarshalMap(result.Item, &alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *DynamoAlertRepository) Update(ctx context.Context, alert *models.Alert) error {
	alert.UpdatedAt = time.Now().UTC()
	item, err := attributevalue.MarshalMap(alert)
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

func (r *DynamoAlertRepository) Delete(ctx context.Context, id string) error {
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.tableName), Key: key})
	return err
}

func (r *DynamoAlertRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error) {
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
	var alerts []*models.Alert
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &alerts); err != nil {
		return nil, nil, err
	}
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return alerts, nextKey, nil
}

func (r *DynamoAlertRepository) ListByByService(ctx context.Context, serviceID string, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("ByService"),
		KeyConditionExpression: aws.String("service_id = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: serviceID},
		},
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var alerts []*models.Alert
	_ = attributevalue.UnmarshalListOfMaps(result.Items, &alerts)
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return alerts, nextKey, nil
}

func (r *DynamoAlertRepository) ListByByTimestamp(ctx context.Context, timestamp string, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("ByTimestamp"),
		KeyConditionExpression: aws.String("timestamp = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: timestamp},
		},
	}
	if limit > 0 {
		input.Limit = aws.Int32(int32(limit))
	}
	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	var alerts []*models.Alert
	_ = attributevalue.UnmarshalListOfMaps(result.Items, &alerts)
	var nextKey map[string]interface{}
	if result.LastEvaluatedKey != nil {
		nextKey = make(map[string]interface{})
		_ = attributevalue.UnmarshalMap(result.LastEvaluatedKey, &nextKey)
	}
	return alerts, nextKey, nil
}
