package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/models"
)

type MockAlertRepository struct {
	alerts map[string]*models.Alert
}

func NewMockAlertRepository() *MockAlertRepository {
	return &MockAlertRepository{
		alerts: make(map[string]*models.Alert),
	}
}

func (r *MockAlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	now := time.Now().UTC()
	alert.CreatedAt = now
	alert.UpdatedAt = now
	key := alert.ID
	r.alerts[key] = alert
	return nil
}
func (r *MockAlertRepository) Get(ctx context.Context, id string) (*models.Alert, error) {
	key := id
	return r.alerts[key], nil
}

func (r *MockAlertRepository) Update(ctx context.Context, alert *models.Alert) error {
	alert.UpdatedAt = time.Now().UTC()
	key := alert.ID
	r.alerts[key] = alert
	return nil
}
func (r *MockAlertRepository) Delete(ctx context.Context, id string) error {
	key := id
	delete(r.alerts, key)
	return nil
}

func (r *MockAlertRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error) {
	var result []*models.Alert
	for _, v := range r.alerts {
		result = append(result, v)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil, nil
}

func (r *MockAlertRepository) ListByByService(ctx context.Context, serviceID string, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error) {
	var result []*models.Alert
	for _, v := range r.alerts {
		if v.ServiceID == serviceID {
			result = append(result, v)
		}
	}
	return result, nil, nil
}

func (r *MockAlertRepository) ListByByTimestamp(ctx context.Context, timestamp string, limit int, lastKey map[string]interface{}) ([]*models.Alert, map[string]interface{}, error) {
	var result []*models.Alert
	for _, v := range r.alerts {
		if v.Timestamp == timestamp {
			result = append(result, v)
		}
	}
	return result, nil, nil
}


func TestAlertRepository_CRUD(t *testing.T) {
	repo := NewMockAlertRepository()
	ctx := context.Background()
	alert := &models.Alert{
		ID: "test-id",
	}
	if err := repo.Create(ctx, alert); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if alert.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	got, err := repo.Get(ctx, "test-id")
	if err != nil || got == nil {
		t.Fatalf("Get: %v", err)
	}
	items, _, _ := repo.List(ctx, 10, nil)
	if len(items) != 1 {
		t.Errorf("List: got %d, want 1", len(items))
	}
	_ = repo.Delete(ctx, "test-id")
	got, _ = repo.Get(ctx, "test-id")
	if got != nil {
		t.Error("Delete failed")
	}
}
