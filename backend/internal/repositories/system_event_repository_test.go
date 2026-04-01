package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/models"
)

type MockSystemEventRepository struct {
	systemEvents map[string]*models.SystemEvent
}

func NewMockSystemEventRepository() *MockSystemEventRepository {
	return &MockSystemEventRepository{
		systemEvents: make(map[string]*models.SystemEvent),
	}
}

func (r *MockSystemEventRepository) Create(ctx context.Context, systemEvent *models.SystemEvent) error {
	now := time.Now().UTC()
	systemEvent.CreatedAt = now
	systemEvent.UpdatedAt = now
	key := systemEvent.ID
	r.systemEvents[key] = systemEvent
	return nil
}
func (r *MockSystemEventRepository) Get(ctx context.Context, id string) (*models.SystemEvent, error) {
	key := id
	return r.systemEvents[key], nil
}

func (r *MockSystemEventRepository) Update(ctx context.Context, systemEvent *models.SystemEvent) error {
	systemEvent.UpdatedAt = time.Now().UTC()
	key := systemEvent.ID
	r.systemEvents[key] = systemEvent
	return nil
}
func (r *MockSystemEventRepository) Delete(ctx context.Context, id string) error {
	key := id
	delete(r.systemEvents, key)
	return nil
}

func (r *MockSystemEventRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error) {
	var result []*models.SystemEvent
	for _, v := range r.systemEvents {
		result = append(result, v)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil, nil
}

func (r *MockSystemEventRepository) ListByByService(ctx context.Context, service string, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error) {
	var result []*models.SystemEvent
	for _, v := range r.systemEvents {
		if v.Service == service {
			result = append(result, v)
		}
	}
	return result, nil, nil
}

func (r *MockSystemEventRepository) ListByByType(ctx context.Context, eventType string, limit int, lastKey map[string]interface{}) ([]*models.SystemEvent, map[string]interface{}, error) {
	var result []*models.SystemEvent
	for _, v := range r.systemEvents {
		if v.Type == eventType {
			result = append(result, v)
		}
	}
	return result, nil, nil
}


func TestSystemEventRepository_CRUD(t *testing.T) {
	repo := NewMockSystemEventRepository()
	ctx := context.Background()
	systemEvent := &models.SystemEvent{
		ID: "test-id",
	}
	if err := repo.Create(ctx, systemEvent); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if systemEvent.CreatedAt.IsZero() {
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
