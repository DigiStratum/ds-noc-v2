package tenant

import (
	"testing"
)

func TestPKPrefix(t *testing.T) {
	tests := []struct {
		name   string
		tenant Tenant
		want   string
	}{
		{
			name:   "user tenant",
			tenant: Tenant{Type: TenantTypeUser, ID: "123"},
			want:   "TENANT#user:123#",
		},
		{
			name:   "org tenant",
			tenant: Tenant{Type: TenantTypeOrg, ID: "my-org"},
			want:   "TENANT#org:my-org#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PKPrefix(tt.tenant); got != tt.want {
				t.Errorf("PKPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPK(t *testing.T) {
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}

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
			name:     "multiple segments",
			segments: []string{"PROJECT", "abc", "MEMBER", "user-1"},
			want:     "TENANT#user:123#PROJECT#abc#MEMBER#user-1",
		},
		{
			name:     "no segments",
			segments: nil,
			want:     "TENANT#user:123#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildPK(tenant, tt.segments...); got != tt.want {
				t.Errorf("BuildPK() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePK(t *testing.T) {
	tests := []struct {
		name         string
		pk           string
		wantTenant   Tenant
		wantSegments []string
		wantErr      bool
	}{
		{
			name:         "user tenant with segments",
			pk:           "TENANT#user:123#ISSUE#456",
			wantTenant:   Tenant{Type: TenantTypeUser, ID: "123"},
			wantSegments: []string{"ISSUE", "456"},
		},
		{
			name:         "org tenant with segments",
			pk:           "TENANT#org:my-org#PROJECT#abc#TASK#1",
			wantTenant:   Tenant{Type: TenantTypeOrg, ID: "my-org"},
			wantSegments: []string{"PROJECT", "abc", "TASK", "1"},
		},
		{
			name:         "tenant only",
			pk:           "TENANT#user:456#",
			wantTenant:   Tenant{Type: TenantTypeUser, ID: "456"},
			wantSegments: nil,
		},
		{
			name:    "missing prefix",
			pk:      "user:123#ISSUE#456",
			wantErr: true,
		},
		{
			name:    "invalid tenant type",
			pk:      "TENANT#team:123#ISSUE#456",
			wantErr: true,
		},
		{
			name:    "empty tenant id",
			pk:      "TENANT#user:#ISSUE#456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, segments, err := ParsePK(tt.pk)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tenant != tt.wantTenant {
				t.Errorf("ParsePK() tenant = %+v, want %+v", tenant, tt.wantTenant)
			}
			if len(segments) != len(tt.wantSegments) {
				t.Errorf("ParsePK() segments = %v, want %v", segments, tt.wantSegments)
				return
			}
			for i := range segments {
				if segments[i] != tt.wantSegments[i] {
					t.Errorf("ParsePK() segments[%d] = %q, want %q", i, segments[i], tt.wantSegments[i])
				}
			}
		})
	}
}

func TestParsePK_RoundTrip(t *testing.T) {
	tenant := Tenant{Type: TenantTypeOrg, ID: "test-org"}
	segments := []string{"ENTITY", "123", "CHILD", "456"}

	pk := BuildPK(tenant, segments...)
	gotTenant, gotSegments, err := ParsePK(pk)
	if err != nil {
		t.Fatalf("ParsePK() error = %v", err)
	}

	if gotTenant != tenant {
		t.Errorf("Round-trip tenant = %+v, want %+v", gotTenant, tenant)
	}
	if len(gotSegments) != len(segments) {
		t.Errorf("Round-trip segments = %v, want %v", gotSegments, segments)
	}
}

func TestValidatePKBelongsToTenant(t *testing.T) {
	tenant1 := Tenant{Type: TenantTypeUser, ID: "user-1"}
	tenant2 := Tenant{Type: TenantTypeUser, ID: "user-2"}

	pk := BuildPK(tenant1, "ISSUE", "123")

	// Should pass for matching tenant
	if err := ValidatePKBelongsToTenant(pk, tenant1); err != nil {
		t.Errorf("ValidatePKBelongsToTenant() should pass for matching tenant: %v", err)
	}

	// Should fail for different tenant
	if err := ValidatePKBelongsToTenant(pk, tenant2); err == nil {
		t.Error("ValidatePKBelongsToTenant() should fail for different tenant")
	}

	// Should fail for malformed pk
	if err := ValidatePKBelongsToTenant("invalid-pk", tenant1); err == nil {
		t.Error("ValidatePKBelongsToTenant() should fail for malformed pk")
	}
}

func TestBuildGSI1PK(t *testing.T) {
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}

	got := BuildGSI1PK(tenant, "ISSUE")
	want := "TENANT#user:123#TYPE#ISSUE"

	if got != want {
		t.Errorf("BuildGSI1PK() = %q, want %q", got, want)
	}
}

func TestBuildGSI1SK(t *testing.T) {
	got := BuildGSI1SK("2024-01-15T10:30:00Z", "issue-123")
	want := "2024-01-15T10:30:00Z#issue-123"

	if got != want {
		t.Errorf("BuildGSI1SK() = %q, want %q", got, want)
	}
}
