package pagination_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// makeContext returns a Gin context with the given raw query string.
func makeContext(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/?"+query, http.NoBody)
	c.Request = req
	return c
}

// ── ParseParams ───────────────────────────────────────────────────────────────

// P1-defaults: no query params → page=1, size=20
func TestParseParams_Defaults(t *testing.T) {
	p := pagination.ParseParams(makeContext(""))
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.Size)
}

// P1-clamp-size: size>100 → 100
func TestParseParams_ClampSize(t *testing.T) {
	p := pagination.ParseParams(makeContext("size=999"))
	assert.Equal(t, 100, p.Size)
}

// P1-clamp-page: page<1 → 1
func TestParseParams_ClampPage(t *testing.T) {
	p := pagination.ParseParams(makeContext("page=0"))
	assert.Equal(t, 1, p.Page)
}

// negative page also clamps to 1
func TestParseParams_NegativePage(t *testing.T) {
	p := pagination.ParseParams(makeContext("page=-5"))
	assert.Equal(t, 1, p.Page)
}

// size=0 falls back to default
func TestParseParams_ZeroSize(t *testing.T) {
	p := pagination.ParseParams(makeContext("size=0"))
	assert.Equal(t, 20, p.Size)
}

// explicit valid values are preserved
func TestParseParams_ValidValues(t *testing.T) {
	p := pagination.ParseParams(makeContext("page=3&size=50"))
	assert.Equal(t, 3, p.Page)
	assert.Equal(t, 50, p.Size)
}

// size exactly 100 is allowed (not clamped)
func TestParseParams_SizeExact100(t *testing.T) {
	p := pagination.ParseParams(makeContext("size=100"))
	assert.Equal(t, 100, p.Size)
}

// non-numeric values fall back to defaults
func TestParseParams_NonNumeric(t *testing.T) {
	p := pagination.ParseParams(makeContext("page=abc&size=xyz"))
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.Size)
}

// ── Offset / Limit ─────────────────────────────────────────────────────────────

func TestParams_OffsetAndLimit(t *testing.T) {
	tests := []struct {
		page       int
		size       int
		wantOffset int
		wantLimit  int
	}{
		{page: 1, size: 20, wantOffset: 0, wantLimit: 20},
		{page: 2, size: 20, wantOffset: 20, wantLimit: 20},
		{page: 3, size: 10, wantOffset: 20, wantLimit: 10},
		{page: 5, size: 100, wantOffset: 400, wantLimit: 100},
	}
	for _, tt := range tests {
		p := pagination.Params{Page: tt.page, Size: tt.size}
		assert.Equal(t, tt.wantOffset, p.Offset(), "page=%d size=%d offset", tt.page, tt.size)
		assert.Equal(t, tt.wantLimit, p.Limit(), "page=%d size=%d limit", tt.page, tt.size)
	}
}

// ── NewPage ────────────────────────────────────────────────────────────────────

// P1-envelope: 137 records, page=2, size=20 → totalPages=7
func TestNewPage_Envelope(t *testing.T) {
	items := make([]string, 20)
	p := pagination.Params{Page: 2, Size: 20}
	pg := pagination.NewPage(items, 137, p)

	assert.Equal(t, 137, int(pg.Total))
	assert.Equal(t, 2, pg.Page)
	assert.Equal(t, 20, pg.Size)
	assert.Equal(t, 7, pg.TotalPages)
	assert.Len(t, pg.Items, 20)
}

// P1-empty: 0 records → items=[], totalPages=0
func TestNewPage_Empty(t *testing.T) {
	p := pagination.Params{Page: 1, Size: 20}
	pg := pagination.NewPage([]string{}, 0, p)

	assert.Equal(t, int64(0), pg.Total)
	assert.Equal(t, 0, pg.TotalPages)
	assert.NotNil(t, pg.Items)
	assert.Len(t, pg.Items, 0)
}

// nil items are coerced to empty slice (never serialize null)
func TestNewPage_NilItems(t *testing.T) {
	p := pagination.Params{Page: 1, Size: 20}
	pg := pagination.NewPage[string](nil, 0, p)
	assert.NotNil(t, pg.Items)
}

// exact ceiling: 100 records, size=20 → totalPages=5 (no remainder)
func TestNewPage_ExactPages(t *testing.T) {
	p := pagination.Params{Page: 1, Size: 20}
	pg := pagination.NewPage(make([]int, 20), 100, p)
	assert.Equal(t, 5, pg.TotalPages)
}

// ceiling: 101 records, size=20 → totalPages=6
func TestNewPage_CeilingPages(t *testing.T) {
	p := pagination.Params{Page: 1, Size: 20}
	pg := pagination.NewPage(make([]int, 20), 101, p)
	assert.Equal(t, 6, pg.TotalPages)
}
