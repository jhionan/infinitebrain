package compliance_test

import (
	"testing"

	"github.com/rian/infinite_brain/pkg/compliance"
)

func TestIsPHI_ReturnsTrueOnlyForPHI(t *testing.T) {
	tests := []struct {
		dc   compliance.DataClass
		want bool
	}{
		{compliance.DataClassPublic, false},
		{compliance.DataClassInternal, false},
		{compliance.DataClassConfidential, false},
		{compliance.DataClassPHI, true},
	}
	for _, tt := range tests {
		if got := compliance.IsPHI(tt.dc); got != tt.want {
			t.Errorf("IsPHI(%v) = %v, want %v", tt.dc, got, tt.want)
		}
	}
}

func TestRequiresEncryption_TrueForConfidentialAndAbove(t *testing.T) {
	tests := []struct {
		dc   compliance.DataClass
		want bool
	}{
		{compliance.DataClassPublic, false},
		{compliance.DataClassInternal, false},
		{compliance.DataClassConfidential, true},
		{compliance.DataClassPHI, true},
	}
	for _, tt := range tests {
		if got := compliance.RequiresEncryption(tt.dc); got != tt.want {
			t.Errorf("RequiresEncryption(%v) = %v, want %v", tt.dc, got, tt.want)
		}
	}
}
