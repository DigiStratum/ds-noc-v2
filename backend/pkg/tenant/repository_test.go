package tenant

import (
	"testing"
)

func TestBaseRepository_PK(t *testing.T) {
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}
	repo := NewBaseRepository(tenant)

	tests := []struct {
		name     string
		segments []string
		want     string
	}{
		{
			name:     "single segment",
			segments: []string{"ISSUE"},
			want:     "TENANT#user:123#ISSUE",
		},
		{
			name:     "two segments",
			segments: []string{"ISSUE", "456"},
			want:     "TENANT#user:123#ISSUE#456",
		},
		{
			name:     "nested segments",
			segments: []string{"PROJECT", "abc", "MEMBER", "user-1"},
			want:     "TENANT#user:123#PROJECT#abc#MEMBER#user-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.PK(tt.segments...)
			if got != tt.want {
				t.Errorf("PK() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBaseRepository_GSI1PK(t *testing.T) {
	tenant := Tenant{Type: TenantTypeOrg, ID: "my-org"}
	repo := NewBaseRepository(tenant)

	got := repo.GSI1PK("ISSUE")
	want := "TENANT#org:my-org#TYPE#ISSUE"

	if got != want {
		t.Errorf("GSI1PK() = %q, want %q", got, want)
	}
}

func TestBaseRepository_ValidatePK(t *testing.T) {
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}
	repo := NewBaseRepository(tenant)

	// Valid PK
	validPK := "TENANT#user:123#ISSUE#456"
	if err := repo.ValidatePK(validPK); err != nil {
		t.Errorf("ValidatePK() should pass for matching tenant: %v", err)
	}

	// Wrong tenant
	wrongTenantPK := "TENANT#user:456#ISSUE#456"
	if err := repo.ValidatePK(wrongTenantPK); err == nil {
		t.Error("ValidatePK() should fail for different tenant")
	}

	// Invalid format
	invalidPK := "invalid-pk"
	if err := repo.ValidatePK(invalidPK); err == nil {
		t.Error("ValidatePK() should fail for invalid format")
	}
}

func TestBaseRepository_GetTenant(t *testing.T) {
	tenant := Tenant{Type: TenantTypeOrg, ID: "test-org"}
	repo := NewBaseRepository(tenant)

	got := repo.GetTenant()
	if got != tenant {
		t.Errorf("GetTenant() = %+v, want %+v", got, tenant)
	}
}

func TestNewScopedQuery(t *testing.T) {
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}
	q := NewScopedQuery(tenant, "my-table", "ISSUE")

	if q.TableName != "my-table" {
		t.Errorf("TableName = %q, want %q", q.TableName, "my-table")
	}
	if q.PKPrefix != "TENANT#user:123#ISSUE" {
		t.Errorf("PKPrefix = %q, want %q", q.PKPrefix, "TENANT#user:123#ISSUE")
	}
	if q.IndexName != "" {
		t.Errorf("IndexName should be empty for primary index, got %q", q.IndexName)
	}
}

func TestScopedQuery_UseGSI(t *testing.T) {
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}
	q := NewScopedQuery(tenant, "my-table", "ISSUE")
	q.UseGSI("gsi1", "PROJECT")

	if q.IndexName != "gsi1" {
		t.Errorf("IndexName = %q, want %q", q.IndexName, "gsi1")
	}
	if q.PKPrefix != "TENANT#user:123#TYPE#PROJECT" {
		t.Errorf("PKPrefix after UseGSI = %q, want %q", q.PKPrefix, "TENANT#user:123#TYPE#PROJECT")
	}
}

func TestEnsureTenantMatch(t *testing.T) {
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}

	tests := []struct {
		name    string
		pk      string
		wantErr bool
	}{
		{
			name:    "matching tenant",
			pk:      "TENANT#user:123#ISSUE#456",
			wantErr: false,
		},
		{
			name:    "different user id",
			pk:      "TENANT#user:456#ISSUE#456",
			wantErr: true,
		},
		{
			name:    "different tenant type",
			pk:      "TENANT#org:123#ISSUE#456",
			wantErr: true,
		},
		{
			name:    "invalid pk format",
			pk:      "invalid",
			wantErr: true,
		},
		{
			name:    "missing tenant prefix",
			pk:      "ISSUE#456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureTenantMatch(tenant, tt.pk)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureTenantMatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRepositoryIsolation verifies that repositories are properly isolated
func TestRepositoryIsolation(t *testing.T) {
	tenant1 := Tenant{Type: TenantTypeUser, ID: "user-1"}
	tenant2 := Tenant{Type: TenantTypeUser, ID: "user-2"}

	repo1 := NewBaseRepository(tenant1)
	repo2 := NewBaseRepository(tenant2)

	pk1 := repo1.PK("ISSUE", "123")
	pk2 := repo2.PK("ISSUE", "123")

	// Same entity ID, different PKs
	if pk1 == pk2 {
		t.Error("Different tenants should produce different PKs")
	}

	// Cross-tenant validation should fail
	if err := repo1.ValidatePK(pk2); err == nil {
		t.Error("repo1 should reject pk2 (different tenant)")
	}
	if err := repo2.ValidatePK(pk1); err == nil {
		t.Error("repo2 should reject pk1 (different tenant)")
	}

	// Same-tenant validation should pass
	if err := repo1.ValidatePK(pk1); err != nil {
		t.Errorf("repo1 should accept its own pk: %v", err)
	}
}
