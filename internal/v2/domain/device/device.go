// Package device contains the domain model for devices.
package device

import "github.com/google/uuid"

const (
	// DefaultPageSize is the page size used when the client does not request
	// one.
	DefaultPageSize = 20

	// MaxPageSize is the largest page size a client can request.
	MaxPageSize = 50
)

// Paging describes a page of a list result.
type Paging struct {
	// Page is a zero-based page index.
	Page uint64

	// Size is the number of items per page.
	Size uint64
}

// Offset converts the paging to a SQL offset.
func (p Paging) Offset() uint64 {
	return p.Page * p.Size
}

// Device is a ship device owned by a user.
type Device struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	SecurityKey string
}
