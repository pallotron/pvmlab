package distro

import (
	"testing"
)

func TestNewExtractor(t *testing.T) {
	tests := []struct {
		name        string
		distroName  string
		wantType    string
		expectError bool
	}{
		{
			name:        "ubuntu extractor",
			distroName:  "ubuntu",
			wantType:    "*distro.UbuntuExtractor",
			expectError: false,
		},
		{
			name:        "fedora extractor",
			distroName:  "fedora",
			wantType:    "*distro.FedoraExtractor",
			expectError: false,
		},
		{
			name:        "unsupported distro",
			distroName:  "debian",
			wantType:    "",
			expectError: true,
		},
		{
			name:        "empty distro name",
			distroName:  "",
			wantType:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := NewExtractor(tt.distroName)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for distro '%s', but got none", tt.distroName)
				}
				if extractor != nil {
					t.Errorf("expected nil extractor for unsupported distro '%s', but got %T", tt.distroName, extractor)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for distro '%s': %v", tt.distroName, err)
				}
				if extractor == nil {
					t.Errorf("expected non-nil extractor for distro '%s'", tt.distroName)
				}
			}
		})
	}
}

func TestNewExtractor_Ubuntu(t *testing.T) {
	extractor, err := NewExtractor("ubuntu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := extractor.(*UbuntuExtractor); !ok {
		t.Errorf("expected *UbuntuExtractor, got %T", extractor)
	}
}

func TestNewExtractor_Fedora(t *testing.T) {
	extractor, err := NewExtractor("fedora")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := extractor.(*FedoraExtractor); !ok {
		t.Errorf("expected *FedoraExtractor, got %T", extractor)
	}
}
