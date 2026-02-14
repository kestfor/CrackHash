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
			name:     "split alphabet into 3 parts with remainder (duplicates allowed)",
			alphabet: "abcd",
			parts:    3,
			want:     []string{"ab", "cd", "ab"}, // ceil(4/3)=2, last part wraps around
		},
		{
			name:     "split 36 chars into 10 parts",
			alphabet: "abcdefghijklmnopqrstuvwxyz0123456789",
			parts:    10,
			want: []string{
				"abcd", "efgh", "ijkl", "mnop", "qrst",
				"uvwx", "yz01", "2345", "6789", "abcd",
			}, // ceil(36/10)=4
		},
		{
			name:     "split 9 chars into 10 parts (more parts than chars)",
			alphabet: "abcdefghi",
			parts:    10,
			want:     []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "a"}, // ceil(9/10)=1
		},
		{
			name:     "empty alphabet",
			alphabet: "",
			parts:    3,
			want:     []string{},
		},
		{
			name:     "zero parts",
			alphabet: "abcd",
			parts:    0,
			want:     []string{},
		},
		{
			name:     "split 26 chars into 5 parts",
			alphabet: "abcdefghijklmnopqrstuvwxyz",
			parts:    5,
			want:     []string{"abcdef", "ghijkl", "mnopqr", "stuvwx", "yz"}, // ceil(26/5)=6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitAlphabet(tt.alphabet, tt.parts)
			assert.Equal(t, tt.want, got, fmt.Sprintf("SplitAlphabet() = %v, want %v", got, tt.want))
		})
	}

}
