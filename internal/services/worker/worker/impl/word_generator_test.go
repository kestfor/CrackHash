package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWordGenerator_Iterate(t *testing.T) {
	alpthabet := "abc"
	maxLength := 2
	variants := []string{
		"a",
		"b",
		"c",
		"aa",
		"ab",
		"ac",
		"ba",
		"bb",
		"bc",
		"ca",
		"cb",
		"cc",
	}

	generator := WordGenerator(maxLength, alpthabet)
	got := make(map[string]bool)
	for word := range generator.Iterate() {
		got[word] = true
	}

	assert.Equal(t, len(variants), len(got))

	for _, variant := range variants {
		assert.True(t, got[variant], "variant %s not found", variant)
	}
}
