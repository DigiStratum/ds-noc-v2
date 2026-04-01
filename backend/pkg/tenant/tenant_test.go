package tenant

import (
	"context"
	"testing"
)

func TestTenant_String(t *testing.T) {
	tests := []struct {
		name     string
		tenant   Tenant
		expected string
	}{
		{
			name:     "user tenant",
			tenant:   Tenant{Type: TenantTypeUser, ID: "123"},
			expected: "user:123",
		},
		{
			name:     "org tenant",
			tenant:   Tenant{Type: TenantTypeOrg, ID: "abc-def-ghi"},
			expected: "org:abc-def-ghi",
		},
		{
			name:     "user tenant with uuid",
			tenant:   Tenant{Type: TenantTypeUser, ID: "550e8400-e29b-41d4-a716-446655440000"},
			expected: "user:550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tenant.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Tenant
		wantErr error
	}{
		{
			name:  "valid user tenant",
			input: "user:123",
			want:  Tenant{Type: TenantTypeUser, ID: "123"},
		},
		{
			name:  "valid org tenant",
			input: "org:abc-def",
			want:  Tenant{Type: TenantTypeOrg, ID: "abc-def"},
		},
		{
			name:  "valid tenant with special chars",
			input: "user:test@example.com",
			want:  Tenant{Type: TenantTypeUser, ID: "test@example.com"},
		},
		{
			name:  "valid tenant with multiple colons",
			input: "org:some:complex:id",
			want:  Tenant{Type: TenantTypeOrg, ID: "some:complex:id"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "no colon",
			input:   "useronly",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "invalid type",
			input:   "team:123",
			wantErr: ErrInvalidType,
		},
		{
			name:    "empty id",
			input:   "user:",
			wantErr: ErrEmptyID,
		},
		{
			name:    "colon only",
			input:   ":",
			wantErr: ErrInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Parse(%q) error = nil, want %v", tt.input, tt.wantErr)
				} else if err != tt.wantErr {
					t.Errorf("Parse(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse(%q) error = %v, want nil", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_RoundTrip(t *testing.T) {
	tenants := []Tenant{
		{Type: TenantTypeUser, ID: "123"},
		{Type: TenantTypeOrg, ID: "my-org"},
		{Type: TenantTypeUser, ID: "uuid-with-dashes-here"},
	}

	for _, original := range tenants {
		str := original.String()
		parsed, err := Parse(str)
		if err != nil {
			t.Errorf("Parse(%q) error = %v", str, err)
			continue
		}
		if parsed != original {
			t.Errorf("Round-trip failed: %+v -> %q -> %+v", original, str, parsed)
		}
	}
}

func TestMustParse(t *testing.T) {
	// Valid parse should not panic
	tenant := MustParse("user:123")
	if tenant.Type != TenantTypeUser || tenant.ID != "123" {
		t.Errorf("MustParse returned wrong tenant: %+v", tenant)
	}

	// Invalid parse should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse should panic on invalid input")
		}
	}()
	MustParse("invalid")
}

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		tenantType TenantType
		id         string
		wantErr    error
	}{
		{
			name:       "valid user",
			tenantType: TenantTypeUser,
			id:         "123",
			wantErr:    nil,
		},
		{
			name:       "valid org",
			tenantType: TenantTypeOrg,
			id:         "my-org",
			wantErr:    nil,
		},
		{
			name:       "invalid type",
			tenantType: "team",
			id:         "123",
			wantErr:    ErrInvalidType,
		},
		{
			name:       "empty id",
			tenantType: TenantTypeUser,
			id:         "",
			wantErr:    ErrEmptyID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.tenantType, tt.id)
			if err != tt.wantErr {
				t.Errorf("New() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewUser(t *testing.T) {
	tenant, err := NewUser("user-123")
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if tenant.Type != TenantTypeUser {
		t.Errorf("NewUser() type = %v, want user", tenant.Type)
	}
	if tenant.ID != "user-123" {
		t.Errorf("NewUser() id = %v, want user-123", tenant.ID)
	}
}

func TestNewOrg(t *testing.T) {
	tenant, err := NewOrg("org-456")
	if err != nil {
		t.Fatalf("NewOrg() error = %v", err)
	}
	if tenant.Type != TenantTypeOrg {
		t.Errorf("NewOrg() type = %v, want org", tenant.Type)
	}
	if tenant.ID != "org-456" {
		t.Errorf("NewOrg() id = %v, want org-456", tenant.ID)
	}
}

func TestTenant_IsZero(t *testing.T) {
	tests := []struct {
		name   string
		tenant Tenant
		want   bool
	}{
		{name: "zero tenant", tenant: Tenant{}, want: true},
		{name: "empty type", tenant: Tenant{ID: "123"}, want: true},
		{name: "empty id", tenant: Tenant{Type: TenantTypeUser}, want: true},
		{name: "valid tenant", tenant: Tenant{Type: TenantTypeUser, ID: "123"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tenant.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTenant_IsUserAndIsOrg(t *testing.T) {
	user := Tenant{Type: TenantTypeUser, ID: "123"}
	org := Tenant{Type: TenantTypeOrg, ID: "456"}

	if !user.IsUser() {
		t.Error("user tenant should return IsUser() = true")
	}
	if user.IsOrg() {
		t.Error("user tenant should return IsOrg() = false")
	}
	if org.IsUser() {
		t.Error("org tenant should return IsUser() = false")
	}
	if !org.IsOrg() {
		t.Error("org tenant should return IsOrg() = true")
	}
}

func TestTenantType_Valid(t *testing.T) {
	tests := []struct {
		tt   TenantType
		want bool
	}{
		{TenantTypeUser, true},
		{TenantTypeOrg, true},
		{"team", false},
		{"", false},
		{"USER", false}, // case-sensitive
	}

	for _, tc := range tests {
		if got := tc.tt.Valid(); got != tc.want {
			t.Errorf("TenantType(%q).Valid() = %v, want %v", tc.tt, got, tc.want)
		}
	}
}

func TestContext_SetGetTenant(t *testing.T) {
	ctx := context.Background()
	tenant := Tenant{Type: TenantTypeUser, ID: "123"}

	// Initially empty
	got := GetTenant(ctx)
	if !got.IsZero() {
		t.Errorf("GetTenant on empty context = %+v, want zero", got)
	}

	// After setting
	ctx = SetTenant(ctx, tenant)
	got = GetTenant(ctx)
	if got != tenant {
		t.Errorf("GetTenant after Set = %+v, want %+v", got, tenant)
	}
}

func TestContext_RequireTenant(t *testing.T) {
	ctx := context.Background()

	// Without tenant - should error
	_, err := RequireTenant(ctx)
	if err != ErrNoTenantInContext {
		t.Errorf("RequireTenant on empty context error = %v, want ErrNoTenantInContext", err)
	}

	// With tenant - should succeed
	tenant := Tenant{Type: TenantTypeOrg, ID: "my-org"}
	ctx = SetTenant(ctx, tenant)
	got, err := RequireTenant(ctx)
	if err != nil {
		t.Errorf("RequireTenant with tenant error = %v", err)
	}
	if got != tenant {
		t.Errorf("RequireTenant with tenant = %+v, want %+v", got, tenant)
	}
}

func TestContext_TenantIsolation(t *testing.T) {
	// Ensure setting tenant in one context doesn't affect another
	ctx1 := context.Background()
	ctx2 := context.Background()

	tenant1 := Tenant{Type: TenantTypeUser, ID: "user-1"}
	tenant2 := Tenant{Type: TenantTypeOrg, ID: "org-1"}

	ctx1 = SetTenant(ctx1, tenant1)
	ctx2 = SetTenant(ctx2, tenant2)

	if got := GetTenant(ctx1); got != tenant1 {
		t.Errorf("ctx1 tenant = %+v, want %+v", got, tenant1)
	}
	if got := GetTenant(ctx2); got != tenant2 {
		t.Errorf("ctx2 tenant = %+v, want %+v", got, tenant2)
	}
}
