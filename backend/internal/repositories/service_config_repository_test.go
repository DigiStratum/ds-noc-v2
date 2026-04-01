package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/models"
)

type MockServiceConfigRepository struct {
	serviceConfigs map[string]*models.ServiceConfig
}

func NewMockServiceConfigRepository() *MockServiceConfigRepository {
	return &MockServiceConfigRepository{
		serviceConfigs: make(map[string]*models.ServiceConfig),
	}
}

func (r *MockServiceConfigRepository) Create(ctx context.Context, serviceConfig *models.ServiceConfig) error {
	now := time.Now().UTC()
	serviceConfig.CreatedAt = now
	serviceConfig.UpdatedAt = now
	key := serviceConfig.ID
	r.serviceConfigs[key] = serviceConfig
	return nil
}
func (r *MockServiceConfigRepository) Get(ctx context.Context, id string) (*models.ServiceConfig, error) {
	key := id
	return r.serviceConfigs[key], nil
}

func (r *MockServiceConfigRepository) Update(ctx context.Context, serviceConfig *models.ServiceConfig) error {
	serviceConfig.UpdatedAt = time.Now().UTC()
	key := serviceConfig.ID
	r.serviceConfigs[key] = serviceConfig
	return nil
}
func (r *MockServiceConfigRepository) Delete(ctx context.Context, id string) error {
	key := id
	delete(r.serviceConfigs, key)
	return nil
}

func (r *MockServiceConfigRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.ServiceConfig, map[string]interface{}, error) {
	var result []*models.ServiceConfig
	for _, v := range r.serviceConfigs {
		result = append(result, v)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil, nil
}

func (r *MockServiceConfigRepository) ListByByName(ctx context.Context, name string, limit int, lastKey map[string]interface{}) ([]*models.ServiceConfig, map[string]interface{}, error) {
	var result []*models.ServiceConfig
	for _, v := range r.serviceConfigs {
		if v.Name == name {
			result = append(result, v)
		}
	}
	return result, nil, nil
}


func TestServiceConfigRepository_CRUD(t *testing.T) {
	repo := NewMockServiceConfigRepository()
	ctx := context.Background()
	serviceConfig := &models.ServiceConfig{
		ID: "test-id",
	}
	if err := repo.Create(ctx, serviceConfig); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if serviceConfig.CreatedAt.IsZero() {
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
