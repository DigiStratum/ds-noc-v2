package models

import "time"

type Alert struct {
	ID             string    `dynamodbav:"id" json:"id"`
	ServiceID      string    `dynamodbav:"service_id" json:"service_id"`
	ServiceName    string    `dynamodbav:"service_name" json:"service_name"`
	Timestamp      string    `dynamodbav:"timestamp" json:"timestamp"`
	Type           string    `dynamodbav:"type" json:"type"`           // recovery, outage, degradation, change
	Severity       string    `dynamodbav:"severity" json:"severity"`   // critical, warning, info
	PreviousStatus string    `dynamodbav:"previous_status" json:"previous_status"`
	CurrentStatus  string    `dynamodbav:"current_status" json:"current_status"`
	Message        string    `dynamodbav:"message" json:"message"`
	LatencyMs      int       `dynamodbav:"latency_ms,omitempty" json:"latency_ms,omitempty"`
	CreatedAt      time.Time `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt      time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

func (m *Alert) TableName() string { return "alert" }

func (m *Alert) GetKey() map[string]interface{} {
	key := map[string]interface{}{"id": m.ID}
	return key
}
