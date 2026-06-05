package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenCodigo verifies that genCodigo produces codes of the correct length,
// uses only uppercase alphanumeric characters, and produces unique values across calls.
func TestGenCodigo(t *testing.T) {
	t.Run("length is 13", func(t *testing.T) {
		code, err := genCodigo()
		require.NoError(t, err)
		assert.Len(t, code, 13, "genCodigo must produce exactly 13 characters")
	})

	t.Run("charset is uppercase alphanumeric (base32 no-padding)", func(t *testing.T) {
		code, err := genCodigo()
		require.NoError(t, err)
		for _, r := range code {
			assert.True(t, (r >= 'A' && r <= 'Z') || (r >= '2' && r <= '7'),
				"genCodigo must use base32 charset (A-Z, 2-7); got %c", r)
		}
	})

	t.Run("two calls produce different codes", func(t *testing.T) {
		code1, err1 := genCodigo()
		require.NoError(t, err1)
		code2, err2 := genCodigo()
		require.NoError(t, err2)
		assert.NotEqual(t, code1, code2, "two calls to genCodigo must produce different values")
	})
}
