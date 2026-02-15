package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestToPtr(t *testing.T) {
	t.Run("int value", func(t *testing.T) {
		val := 42
		ptr := ToPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("string value", func(t *testing.T) {
		val := "hello"
		ptr := ToPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("zero value", func(t *testing.T) {
		val := 0
		ptr := ToPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("struct value", func(t *testing.T) {
		type testStruct struct {
			Name string
			Age  int
		}
		val := testStruct{Name: "test", Age: 25}
		ptr := ToPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("bool value", func(t *testing.T) {
		val := true
		ptr := ToPtr(val)
		assert.NotNil(t, ptr)
		assert.Equal(t, val, *ptr)
	})

	t.Run("modifying original does not affect pointer", func(t *testing.T) {
		val := 10
		ptr := ToPtr(val)
		val = 20
		assert.Equal(t, 10, *ptr)
	})
}
