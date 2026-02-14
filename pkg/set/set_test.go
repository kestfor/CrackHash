package set

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	s := New[int]()
	assert.NotNil(t, s)
	assert.Equal(t, 0, s.Size())
}

func TestSet_Add(t *testing.T) {
	tests := []struct {
		name     string
		items    []int
		wantSize int
	}{
		{
			name:     "add single item",
			items:    []int{1},
			wantSize: 1,
		},
		{
			name:     "add multiple items",
			items:    []int{1, 2, 3},
			wantSize: 3,
		},
		{
			name:     "add duplicate items",
			items:    []int{1, 1, 1},
			wantSize: 1,
		},
		{
			name:     "add no items",
			items:    []int{},
			wantSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New[int]()
			s.Add(tt.items...)
			assert.Equal(t, tt.wantSize, s.Size())
		})
	}
}

func TestSet_Remove(t *testing.T) {
	tests := []struct {
		name       string
		initial    []int
		toRemove   int
		wantSize   int
		shouldHave []int
	}{
		{
			name:       "remove existing item",
			initial:    []int{1, 2, 3},
			toRemove:   2,
			wantSize:   2,
			shouldHave: []int{1, 3},
		},
		{
			name:       "remove non-existing item",
			initial:    []int{1, 2, 3},
			toRemove:   5,
			wantSize:   3,
			shouldHave: []int{1, 2, 3},
		},
		{
			name:       "remove from empty set",
			initial:    []int{},
			toRemove:   1,
			wantSize:   0,
			shouldHave: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New[int]()
			s.Add(tt.initial...)
			s.Remove(tt.toRemove)
			assert.Equal(t, tt.wantSize, s.Size())
			for _, item := range tt.shouldHave {
				assert.True(t, s.Contains(item))
			}
		})
	}
}

func TestSet_Contains(t *testing.T) {
	tests := []struct {
		name    string
		initial []int
		check   int
		want    bool
	}{
		{
			name:    "contains existing item",
			initial: []int{1, 2, 3},
			check:   2,
			want:    true,
		},
		{
			name:    "does not contain item",
			initial: []int{1, 2, 3},
			check:   5,
			want:    false,
		},
		{
			name:    "empty set contains nothing",
			initial: []int{},
			check:   1,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New[int]()
			s.Add(tt.initial...)
			assert.Equal(t, tt.want, s.Contains(tt.check))
		})
	}
}

func TestSet_Size(t *testing.T) {
	tests := []struct {
		name     string
		items    []int
		wantSize int
	}{
		{
			name:     "empty set",
			items:    []int{},
			wantSize: 0,
		},
		{
			name:     "set with items",
			items:    []int{1, 2, 3, 4, 5},
			wantSize: 5,
		},
		{
			name:     "set with duplicates",
			items:    []int{1, 1, 2, 2, 3},
			wantSize: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New[int]()
			s.Add(tt.items...)
			assert.Equal(t, tt.wantSize, s.Size())
		})
	}
}

func TestSet_Union(t *testing.T) {
	tests := []struct {
		name string
		set1 []int
		set2 []int
		want []int
	}{
		{
			name: "union of two sets",
			set1: []int{1, 2, 3},
			set2: []int{3, 4, 5},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "union with empty set",
			set1: []int{1, 2, 3},
			set2: []int{},
			want: []int{1, 2, 3},
		},
		{
			name: "union of two empty sets",
			set1: []int{},
			set2: []int{},
			want: []int{},
		},
		{
			name: "union of identical sets",
			set1: []int{1, 2, 3},
			set2: []int{1, 2, 3},
			want: []int{1, 2, 3},
		},
		{
			name: "union of disjoint sets",
			set1: []int{1, 2},
			set2: []int{3, 4},
			want: []int{1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1 := New[int]()
			s1.Add(tt.set1...)
			s2 := New[int]()
			s2.Add(tt.set2...)

			result := s1.Union(s2)
			assert.Equal(t, len(tt.want), result.Size())
			for _, item := range tt.want {
				assert.True(t, result.Contains(item))
			}
		})
	}
}

func TestSet_Intersection(t *testing.T) {
	tests := []struct {
		name string
		set1 []int
		set2 []int
		want []int
	}{
		{
			name: "intersection with common elements",
			set1: []int{1, 2, 3, 4},
			set2: []int{3, 4, 5, 6},
			want: []int{3, 4},
		},
		{
			name: "intersection with empty set",
			set1: []int{1, 2, 3},
			set2: []int{},
			want: []int{},
		},
		{
			name: "intersection of disjoint sets",
			set1: []int{1, 2},
			set2: []int{3, 4},
			want: []int{},
		},
		{
			name: "intersection of identical sets",
			set1: []int{1, 2, 3},
			set2: []int{1, 2, 3},
			want: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1 := New[int]()
			s1.Add(tt.set1...)
			s2 := New[int]()
			s2.Add(tt.set2...)

			result := s1.Intersection(s2)
			assert.Equal(t, len(tt.want), result.Size())
			for _, item := range tt.want {
				assert.True(t, result.Contains(item))
			}
		})
	}
}

func TestSet_Difference(t *testing.T) {
	tests := []struct {
		name string
		set1 []int
		set2 []int
		want []int
	}{
		{
			name: "difference with common elements",
			set1: []int{1, 2, 3, 4},
			set2: []int{3, 4, 5, 6},
			want: []int{1, 2},
		},
		{
			name: "difference with empty set",
			set1: []int{1, 2, 3},
			set2: []int{},
			want: []int{1, 2, 3},
		},
		{
			name: "difference of disjoint sets",
			set1: []int{1, 2},
			set2: []int{3, 4},
			want: []int{1, 2},
		},
		{
			name: "difference of identical sets",
			set1: []int{1, 2, 3},
			set2: []int{1, 2, 3},
			want: []int{},
		},
		{
			name: "difference when first is subset",
			set1: []int{1, 2},
			set2: []int{1, 2, 3, 4},
			want: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1 := New[int]()
			s1.Add(tt.set1...)
			s2 := New[int]()
			s2.Add(tt.set2...)

			result := s1.Difference(s2)
			assert.Equal(t, len(tt.want), result.Size())
			for _, item := range tt.want {
				assert.True(t, result.Contains(item))
			}
		})
	}
}

func TestSet_Slice(t *testing.T) {
	tests := []struct {
		name  string
		items []int
	}{
		{
			name:  "slice from set with items",
			items: []int{3, 1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New[int]()
			s.Add(tt.items...)

			result := s.Slice()
			assert.Equal(t, len(tt.items), len(result))

			// sort both slices for comparison since set order is not guaranteed
			sort.Ints(result)
			expected := make([]int, len(tt.items))
			copy(expected, tt.items)
			sort.Ints(expected)
			assert.Equal(t, expected, result)
		})
	}
}
