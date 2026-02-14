package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
