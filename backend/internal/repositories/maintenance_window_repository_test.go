package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/models"
)

type MockMaintenanceWindowRepository struct {
	maintenanceWindows map[string]*models.MaintenanceWindow
}

func NewMockMaintenanceWindowRepository() *MockMaintenanceWindowRepository {
	return &MockMaintenanceWindowRepository{
		maintenanceWindows: make(map[string]*models.MaintenanceWindow),
	}
}

func (r *MockMaintenanceWindowRepository) Create(ctx context.Context, maintenanceWindow *models.MaintenanceWindow) error {
	now := time.Now().UTC()
	maintenanceWindow.CreatedAt = now
	maintenanceWindow.UpdatedAt = now
	key := maintenanceWindow.ID
	r.maintenanceWindows[key] = maintenanceWindow
	return nil
}
func (r *MockMaintenanceWindowRepository) Get(ctx context.Context, id string) (*models.MaintenanceWindow, error) {
	key := id
	return r.maintenanceWindows[key], nil
}

func (r *MockMaintenanceWindowRepository) Update(ctx context.Context, maintenanceWindow *models.MaintenanceWindow) error {
	maintenanceWindow.UpdatedAt = time.Now().UTC()
	key := maintenanceWindow.ID
	r.maintenanceWindows[key] = maintenanceWindow
	return nil
}
func (r *MockMaintenanceWindowRepository) Delete(ctx context.Context, id string) error {
	key := id
	delete(r.maintenanceWindows, key)
	return nil
}

func (r *MockMaintenanceWindowRepository) List(ctx context.Context, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error) {
	var result []*models.MaintenanceWindow
	for _, v := range r.maintenanceWindows {
		result = append(result, v)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil, nil
}

func (r *MockMaintenanceWindowRepository) ListByByService(ctx context.Context, service string, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error) {
	var result []*models.MaintenanceWindow
	for _, v := range r.maintenanceWindows {
		if v.Service == service {
			result = append(result, v)
		}
	}
	return result, nil, nil
}

func (r *MockMaintenanceWindowRepository) ListByByStartTime(ctx context.Context, startTime string, limit int, lastKey map[string]interface{}) ([]*models.MaintenanceWindow, map[string]interface{}, error) {
	var result []*models.MaintenanceWindow
	for _, v := range r.maintenanceWindows {
		if v.StartTime == startTime {
			result = append(result, v)
		}
	}
	return result, nil, nil
}


func TestMaintenanceWindowRepository_CRUD(t *testing.T) {
	repo := NewMockMaintenanceWindowRepository()
	ctx := context.Background()
	maintenanceWindow := &models.MaintenanceWindow{
		ID: "test-id",
	}
	if err := repo.Create(ctx, maintenanceWindow); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if maintenanceWindow.CreatedAt.IsZero() {
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
