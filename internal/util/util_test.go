package util

import (
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		sizeStr string
		want    int64
		wantErr bool
	}{
		{
			name:    "Gigabytes uppercase",
			sizeStr: "10G",
			want:    10 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "Gigabytes lowercase",
			sizeStr: "10g",
			want:    10 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "Megabytes",
			sizeStr: "512M",
			want:    512 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "Kilobytes",
			sizeStr: "2048K",
			want:    2048 * 1024,
			wantErr: false,
		},
		{
			name:    "Bytes only number",
			sizeStr: "1024",
			want:    1024,
			wantErr: false,
		},
		{
			name:    "Bytes with unit",
			sizeStr: "1024B",
			want:    1024,
			wantErr: false,
		},
		{
			name:    "Invalid format",
			sizeStr: "abc",
			wantErr: true,
		},
		{
			name:    "Invalid unit",
			sizeStr: "10X",
			wantErr: true,
		},
		{
			name:    "Empty string",
			sizeStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSize(tt.sizeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
