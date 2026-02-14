package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSearchSpace(t *testing.T) {
	t.Run("creates search space with correct properties", func(t *testing.T) {
		ss := NewSearchSpace("abc", 3)

		assert.Equal(t, []byte("abc"), ss.alphabet)
		assert.Equal(t, uint64(3), ss.base)
		assert.Equal(t, 3, ss.maxLen)
	})
}

func TestSearchSpace_TotalSize(t *testing.T) {
	tests := []struct {
		name     string
		alphabet string
		maxLen   int
		want     uint64
	}{
		{
			name:     "alphabet size 2, max length 1",
			alphabet: "ab",
			maxLen:   1,
			want:     2, // 2^1 = 2
		},
		{
			name:     "alphabet size 2, max length 2",
			alphabet: "ab",
			maxLen:   2,
			want:     6, // 2 + 4 = 6
		},
		{
			name:     "alphabet size 2, max length 3",
			alphabet: "ab",
			maxLen:   3,
			want:     14, // 2 + 4 + 8 = 14
		},
		{
			name:     "alphabet size 3, max length 2",
			alphabet: "abc",
			maxLen:   2,
			want:     12, // 3 + 9 = 12
		},
		{
			name:     "alphabet size 36, max length 1",
			alphabet: "abcdefghijklmnopqrstuvwxyz0123456789",
			maxLen:   1,
			want:     36,
		},
		{
			name:     "alphabet size 36, max length 2",
			alphabet: "abcdefghijklmnopqrstuvwxyz0123456789",
			maxLen:   2,
			want:     36 + 36*36, // 36 + 1296 = 1332
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := NewSearchSpace(tt.alphabet, tt.maxLen)
			got := ss.TotalSize()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSearchSpace_FillWord(t *testing.T) {
	t.Run("alphabet ab, maxLen 2", func(t *testing.T) {
		// Total space: a, b, aa, ab, ba, bb (6 words)
		// Index mapping:
		// 0 -> a
		// 1 -> b
		// 2 -> aa
		// 3 -> ab
		// 4 -> ba
		// 5 -> bb
		ss := NewSearchSpace("ab", 2)
		buf := make([]byte, 2)

		tests := []struct {
			index   uint64
			want    string
			wantLen int
		}{
			{0, "a", 1},
			{1, "b", 1},
			{2, "aa", 2},
			{3, "ab", 2},
			{4, "ba", 2},
			{5, "bb", 2},
		}

		for _, tt := range tests {
			length := ss.FillWord(tt.index, buf)
			assert.Equal(t, tt.wantLen, length, "index %d", tt.index)
			assert.Equal(t, tt.want, string(buf[:length]), "index %d", tt.index)
		}
	})

	t.Run("alphabet abc, maxLen 2", func(t *testing.T) {
		// Total space: a, b, c, aa, ab, ac, ba, bb, bc, ca, cb, cc (12 words)
		// Index mapping:
		// 0 -> a, 1 -> b, 2 -> c
		// 3 -> aa, 4 -> ab, 5 -> ac
		// 6 -> ba, 7 -> bb, 8 -> bc
		// 9 -> ca, 10 -> cb, 11 -> cc
		ss := NewSearchSpace("abc", 2)
		buf := make([]byte, 2)

		tests := []struct {
			index   uint64
			want    string
			wantLen int
		}{
			{0, "a", 1},
			{1, "b", 1},
			{2, "c", 1},
			{3, "aa", 2},
			{4, "ab", 2},
			{5, "ac", 2},
			{6, "ba", 2},
			{7, "bb", 2},
			{8, "bc", 2},
			{9, "ca", 2},
			{10, "cb", 2},
			{11, "cc", 2},
		}

		for _, tt := range tests {
			length := ss.FillWord(tt.index, buf)
			assert.Equal(t, tt.wantLen, length, "index %d", tt.index)
			assert.Equal(t, tt.want, string(buf[:length]), "index %d", tt.index)
		}
	})

	t.Run("alphabet abc, maxLen 3", func(t *testing.T) {
		// First 3: a, b, c
		// Next 9: aa..cc
		// Last 27: aaa..ccc
		ss := NewSearchSpace("abc", 3)
		buf := make([]byte, 3)

		tests := []struct {
			index   uint64
			want    string
			wantLen int
		}{
			{0, "a", 1},
			{2, "c", 1},
			{3, "aa", 2},
			{11, "cc", 2},
			{12, "aaa", 3},
			{13, "aab", 3},
			{14, "aac", 3},
			{15, "aba", 3},
			{38, "ccc", 3}, // last word: index 38 = 3 + 9 + 27 - 1
		}

		for _, tt := range tests {
			length := ss.FillWord(tt.index, buf)
			assert.Equal(t, tt.wantLen, length, "index %d", tt.index)
			assert.Equal(t, tt.want, string(buf[:length]), "index %d", tt.index)
		}
	})

	t.Run("out of range index returns 0", func(t *testing.T) {
		ss := NewSearchSpace("ab", 2)
		buf := make([]byte, 2)

		// Total size is 6, so index 6 is out of range
		length := ss.FillWord(6, buf)
		assert.Equal(t, 0, length)

		length = ss.FillWord(100, buf)
		assert.Equal(t, 0, length)
	})

	t.Run("single character alphabet", func(t *testing.T) {
		ss := NewSearchSpace("x", 3)
		buf := make([]byte, 3)

		// Total space: x, xx, xxx (3 words)
		tests := []struct {
			index   uint64
			want    string
			wantLen int
		}{
			{0, "x", 1},
			{1, "xx", 2},
			{2, "xxx", 3},
		}

		for _, tt := range tests {
			length := ss.FillWord(tt.index, buf)
			assert.Equal(t, tt.wantLen, length, "index %d", tt.index)
			assert.Equal(t, tt.want, string(buf[:length]), "index %d", tt.index)
		}
	})
}

func TestSearchSpace_Consistency(t *testing.T) {
	t.Run("TotalSize matches actual word count", func(t *testing.T) {
		ss := NewSearchSpace("abc", 3)
		buf := make([]byte, 3)

		totalSize := ss.TotalSize()
		wordCount := uint64(0)

		for i := uint64(0); i < totalSize; i++ {
			length := ss.FillWord(i, buf)
			if length > 0 {
				wordCount++
			}
		}

		assert.Equal(t, totalSize, wordCount)

		// Next index should be out of range
		length := ss.FillWord(totalSize, buf)
		assert.Equal(t, 0, length)
	})

	t.Run("all words are unique", func(t *testing.T) {
		ss := NewSearchSpace("ab", 3)
		buf := make([]byte, 3)

		totalSize := ss.TotalSize()
		words := make(map[string]uint64)

		for i := uint64(0); i < totalSize; i++ {
			length := ss.FillWord(i, buf)
			word := string(buf[:length])

			if prevIdx, exists := words[word]; exists {
				t.Errorf("duplicate word %q at indices %d and %d", word, prevIdx, i)
			}
			words[word] = i
		}

		assert.Equal(t, int(totalSize), len(words))
	})
}
