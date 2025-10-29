package value

import (
	"testing"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewPlan(t *testing.T) {
	tests := []struct {
		name    string
		planStr string
		want    Plan
		wantErr error
	}{
		{
			name:    "valid free plan",
			planStr: "free",
			want:    PlanFree,
			wantErr: nil,
		},
		{
			name:    "valid eco plan",
			planStr: "eco",
			want:    PlanEco,
			wantErr: nil,
		},
		{
			name:    "valid pro plan",
			planStr: "pro",
			want:    PlanPro,
			wantErr: nil,
		},
		{
			name:    "invalid plan",
			planStr: "invalid",
			want:    "",
			wantErr: projecterrors.ErrInvalidPlan,
		},
		{
			name:    "empty plan",
			planStr: "",
			want:    "",
			wantErr: projecterrors.ErrInvalidPlan,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPlan(tt.planStr)
			if err != tt.wantErr {
				t.Errorf("NewPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewPlan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_IsValid(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want bool
	}{
		{"free is valid", PlanFree, true},
		{"eco is valid", PlanEco, true},
		{"pro is valid", PlanPro, true},
		{"invalid plan", Plan("invalid"), false},
		{"empty plan", Plan(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.IsValid(); got != tt.want {
				t.Errorf("Plan.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_IsScaleToZero(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want bool
	}{
		{"free supports scale to zero", PlanFree, true},
		{"eco supports scale to zero", PlanEco, true},
		{"pro does not support scale to zero", PlanPro, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.IsScaleToZero(); got != tt.want {
				t.Errorf("Plan.IsScaleToZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_IsAlwaysOn(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want bool
	}{
		{"free is not always on", PlanFree, false},
		{"eco is not always on", PlanEco, false},
		{"pro is always on", PlanPro, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.IsAlwaysOn(); got != tt.want {
				t.Errorf("Plan.IsAlwaysOn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_HasAdvertisement(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want bool
	}{
		{"free has advertisement", PlanFree, true},
		{"eco does not have advertisement", PlanEco, false},
		{"pro does not have advertisement", PlanPro, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.HasAdvertisement(); got != tt.want {
				t.Errorf("Plan.HasAdvertisement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_IsUsageBased(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want bool
	}{
		{"free is usage based", PlanFree, true},
		{"eco is usage based", PlanEco, true},
		{"pro is not usage based", PlanPro, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.IsUsageBased(); got != tt.want {
				t.Errorf("Plan.IsUsageBased() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_IsFixedPrice(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want bool
	}{
		{"free is not fixed price", PlanFree, false},
		{"eco is not fixed price", PlanEco, false},
		{"pro is fixed price", PlanPro, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.IsFixedPrice(); got != tt.want {
				t.Errorf("Plan.IsFixedPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlan_String(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want string
	}{
		{"free plan string", PlanFree, "free"},
		{"eco plan string", PlanEco, "eco"},
		{"pro plan string", PlanPro, "pro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.String(); got != tt.want {
				t.Errorf("Plan.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
