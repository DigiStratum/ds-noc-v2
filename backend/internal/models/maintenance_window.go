package models

import "time"

type MaintenanceWindow struct {
	ID          string    `dynamodbav:"id" json:"id"`
	Service     string    `dynamodbav:"service" json:"service"`
	StartTime   string    `dynamodbav:"start_time" json:"start_time"`
	EndTime     string    `dynamodbav:"end_time" json:"end_time"`
	Description string    `dynamodbav:"description" json:"description"`
	CreatedAt   time.Time `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt   time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

func (m *MaintenanceWindow) TableName() string { return "maintenance_window" }

func (m *MaintenanceWindow) GetKey() map[string]interface{} {
	key := map[string]interface{}{"id": m.ID}
	return key
}
