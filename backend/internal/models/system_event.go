package models

import "time"

type SystemEvent struct {
	ID        string    `dynamodbav:"id" json:"id"`
	Timestamp string    `dynamodbav:"timestamp" json:"timestamp"`
	Type      string    `dynamodbav:"type" json:"type"`         // deployment, alert, maintenance, config_change
	Severity  string    `dynamodbav:"severity" json:"severity"` // info, warning, error
	Service   string    `dynamodbav:"service" json:"service"`
	Message   string    `dynamodbav:"message" json:"message"`
	User      string    `dynamodbav:"user,omitempty" json:"user,omitempty"`
	CreatedAt time.Time `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

func (m *SystemEvent) TableName() string { return "system_event" }

func (m *SystemEvent) GetKey() map[string]interface{} {
	key := map[string]interface{}{"id": m.ID}
	return key
}
