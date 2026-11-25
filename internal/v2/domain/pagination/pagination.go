// Package pagination defines cursor-based repository pages.
package pagination

import "errors"

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// ErrInvalidLimit indicates that the requested limit is outside the allowed
// range.
var ErrInvalidLimit = errors.New("invalid page limit")

// Request selects items after a cursor. A zero limit uses DefaultLimit.
type Request[C any] struct {
	After *C
	Limit int
}

// Size validates and returns the requested page size.
func (r Request[C]) Size() (int, error) {
	if r.Limit == 0 {
		return DefaultLimit, nil
	}

	if r.Limit < 0 || r.Limit > MaxLimit {
		return 0, ErrInvalidLimit
	}

	return r.Limit, nil
}

// Page contains a page and a cursor for the next page, if one exists.
type Page[T, C any] struct {
	Items []T
	Next  *C
}

// FromRows forms a page from at most size+1 ordered rows.
func FromRows[T, C any](items []T, size int, key func(T) C) Page[T, C] {
	page := Page[T, C]{Items: items, Next: nil}
	if len(items) > size {
		page.Items = items[:size]
		cursor := key(page.Items[size-1])
		page.Next = &cursor
	}

	return page
}
