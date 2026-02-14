package manager

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitAlphabet(t *testing.T) {
	tests := []struct {
		name     string
		alphabet string
		parts    int
		want     []string
	}{
		{
			name:     "split alphabet into 2 parts",
			alphabet: "abcd",
			parts:    2,
			want:     []string{"ab", "cd"},
		},
		{
			name:     "split alphabet into 3 parts",
			alphabet: "abc",
			parts:    3,
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "split alphabet into 1 part",
			alphabet: "abcd",
			parts:    1,
			want:     []string{"abcd"},
		},
		{
			name:     "split alphabet into 3 parts with remainder",
			alphabet: "abcd",
			parts:    3,
			want:     []string{"ab", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitAlphabet(tt.alphabet, tt.parts)
			assert.Equal(t, tt.want, got, fmt.Sprintf("SplitAlphabet() = %v, want %v", got, tt.want))
		})
	}

}
