package models

import "time"

type ServiceConfig struct {
	ID             string    `dynamodbav:"id" json:"id"`
	Name           string    `dynamodbav:"name" json:"name"`
	URL            string    `dynamodbav:"url" json:"url"`
	HealthEndpoint string    `dynamodbav:"health_endpoint" json:"health_endpoint"`
	Critical       bool      `dynamodbav:"critical" json:"critical"`
	CreatedAt      time.Time `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt      time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

func (m *ServiceConfig) TableName() string { return "service_config" }

func (m *ServiceConfig) GetKey() map[string]interface{} {
	key := map[string]interface{}{"id": m.ID}
	return key
}
