package set

import (
	"maps"
	"slices"
)

type Set[T comparable] map[T]struct{}

func New[T comparable]() Set[T] {
	return make(map[T]struct{})
}

func (s Set[T]) Add(item ...T) {
	for _, i := range item {
		s[i] = struct{}{}
	}
}

func (s Set[T]) Remove(item T) {
	delete(s, item)
}

func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

func (s Set[T]) Size() int {
	return len(s)
}

func (s Set[T]) Union(other Set[T]) Set[T] {
	result := New[T]()
	for item := range s {
		result.Add(item)
	}
	for item := range other {
		result.Add(item)
	}
	return result
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := New[T]()
	for item := range s {
		if other.Contains(item) {
			result.Add(item)
		}
	}

	return result
}

func (s Set[T]) Difference(other Set[T]) Set[T] {
	result := New[T]()
	for item := range s {
		if !other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

func (s Set[T]) Slice() []T {
	return slices.Collect(maps.Keys(s))
}
