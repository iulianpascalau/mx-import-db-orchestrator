package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureHTTPSPrefix(t *testing.T) {
	t.Parallel()

	t.Run("empty prefix should not prepend", func(t *testing.T) {
		assert.Equal(t, "", EnsureHTTPSPrefix(""))
	})
	t.Run("existing http prefix should not prepend", func(t *testing.T) {
		assert.Equal(t, "http://aaa.com", EnsureHTTPSPrefix("http://aaa.com"))
	})
	t.Run("existing https prefix should not prepend", func(t *testing.T) {
		assert.Equal(t, "https://aaa.com", EnsureHTTPSPrefix("https://aaa.com"))
	})
	t.Run("should prepend", func(t *testing.T) {
		assert.Equal(t, "https://aaa.com", EnsureHTTPSPrefix("aaa.com"))
		assert.Equal(t, "https://192.168.1.1", EnsureHTTPSPrefix("192.168.1.1"))
	})
}
