package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchSpaceSize(t *testing.T) {
	tests := []struct {
		name         string
		alphabetSize int
		maxLength    int
		want         uint64
	}{
		{
			name:         "alphabet size 3, max length 1",
			alphabetSize: 3,
			maxLength:    1,
			want:         3, // 3^1 = 3
		},
		{
			name:         "alphabet size 3, max length 2",
			alphabetSize: 3,
			maxLength:    2,
			want:         12, // 3 + 9 = 12
		},
		{
			name:         "alphabet size 2, max length 3",
			alphabetSize: 2,
			maxLength:    3,
			want:         14, // 2 + 4 + 8 = 14
		},
		{
			name:         "alphabet size 36, max length 1",
			alphabetSize: 36,
			maxLength:    1,
			want:         36,
		},
		{
			name:         "alphabet size 36, max length 2",
			alphabetSize: 36,
			maxLength:    2,
			want:         36 + 36*36, // 36 + 1296 = 1332
		},
		{
			name:         "zero alphabet size",
			alphabetSize: 0,
			maxLength:    5,
			want:         0,
		},
		{
			name:         "zero max length",
			alphabetSize: 36,
			maxLength:    0,
			want:         0,
		},
		{
			name:         "negative values",
			alphabetSize: -1,
			maxLength:    -1,
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SearchSpaceSize(tt.alphabetSize, tt.maxLength)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitRange(t *testing.T) {
	tests := []struct {
		name      string
		totalSize uint64
		parts     int
		want      []Range
		wantErr   bool
	}{
		{
			name:      "split 10 into 2 parts",
			totalSize: 10,
			parts:     2,
			want:      []Range{{0, 5}, {5, 10}},
			wantErr:   false,
		},
		{
			name:      "split 10 into 3 parts",
			totalSize: 10,
			parts:     3,
			want:      []Range{{0, 4}, {4, 7}, {7, 10}}, // 4 + 3 + 3 = 10
			wantErr:   false,
		},
		{
			name:      "split 100 into 3 parts",
			totalSize: 100,
			parts:     3,
			want:      []Range{{0, 34}, {34, 67}, {67, 100}}, // 34 + 33 + 33 = 100
			wantErr:   false,
		},
		{
			name:      "split 12 into 4 parts",
			totalSize: 12,
			parts:     4,
			want:      []Range{{0, 3}, {3, 6}, {6, 9}, {9, 12}},
			wantErr:   false,
		},
		{
			name:      "split 5 into 5 parts",
			totalSize: 5,
			parts:     5,
			want:      []Range{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}},
			wantErr:   false,
		},
		{
			name:      "split into 1 part",
			totalSize: 100,
			parts:     1,
			want:      []Range{{0, 100}},
			wantErr:   false,
		},
		{
			name:      "zero parts",
			totalSize: 100,
			parts:     0,
			want:      []Range{},
			wantErr:   false,
		},
		{
			name:      "zero total size",
			totalSize: 0,
			parts:     3,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "parts exceed total size",
			totalSize: 3,
			parts:     5,
			want:      nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitRange(tt.totalSize, tt.parts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			// Verify ranges are contiguous and cover the entire space
			if len(got) > 0 {
				assert.Equal(t, uint64(0), got[0].Start)
				assert.Equal(t, tt.totalSize, got[len(got)-1].End)
				for i := 1; i < len(got); i++ {
					assert.Equal(t, got[i-1].End, got[i].Start, "ranges should be contiguous")
				}
			}
		})
	}
}
