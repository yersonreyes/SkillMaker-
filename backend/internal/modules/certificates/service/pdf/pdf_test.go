package pdf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderCertificate verifies that RenderCertificate returns non-empty bytes
// and no error. The byte content is opaque (PDF binary); we assert structural
// correctness only (non-nil, non-empty).
func TestRenderCertificate(t *testing.T) {
	nombre := "Ana López"
	titulo := "Go Avanzado"
	fecha := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	codigo := "ABCD1234EFG"

	got, err := RenderCertificate(nombre, titulo, fecha, codigo)

	require.NoError(t, err, "RenderCertificate must not return an error")
	assert.NotEmpty(t, got, "RenderCertificate must return non-empty bytes")
	assert.Greater(t, len(got), 100, "RenderCertificate must return at least 100 bytes (minimum PDF size)")
}
