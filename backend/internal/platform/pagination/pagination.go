// Package pagination provides a reusable pagination primitive for Gin handlers.
// It parses ?page and ?size query parameters with clamping, exposes Offset/Limit
// helpers for GORM queries, and builds the JSON envelope returned to clients.
package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultSize = 20
	maxSize     = 100
)

// Params holds clamped pagination inputs. Page is 1-based.
type Params struct {
	Page int // >= 1
	Size int // 1..100
}

// ParseParams reads ?page and ?size from a Gin context, applying defaults and clamps:
//
//	page < 1  → 1
//	size < 1  → 20 (default)
//	size > 100 → 100
func ParseParams(c *gin.Context) Params {
	page := atoiDefault(c.Query("page"), 1)
	size := atoiDefault(c.Query("size"), defaultSize)

	if page < 1 {
		page = 1
	}

	if size < 1 {
		size = defaultSize
	}

	if size > maxSize {
		size = maxSize
	}

	return Params{Page: page, Size: size}
}

// Offset returns the number of rows to skip for GORM's .Offset() call.
func (p Params) Offset() int { return (p.Page - 1) * p.Size }

// Limit returns the page size for GORM's .Limit() call.
func (p Params) Limit() int { return p.Size }

// Page is the generic JSON envelope returned to clients.
// JSON field names follow camelCase per the project convention.
type Page[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// NewPage builds the envelope from a slice of items, the total filtered count,
// and the pagination params used for this query. It computes totalPages via
// integer ceiling division and ensures Items is never nil (serialises as []).
func NewPage[T any](items []T, total int64, p Params) Page[T] {
	if items == nil {
		items = []T{}
	}

	totalPages := 0
	if p.Size > 0 {
		totalPages = int((total + int64(p.Size) - 1) / int64(p.Size))
	}

	return Page[T]{
		Items:      items,
		Page:       p.Page,
		Size:       p.Size,
		Total:      total,
		TotalPages: totalPages,
	}
}

// atoiDefault converts s to int; returns def if s is empty or non-numeric.
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}

	return n
}
