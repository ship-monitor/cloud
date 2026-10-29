package problem

import (
	"fmt"
	"strings"
)

type Error struct {
	Type   string
	Title  string
	Status int
	Detail string
}

func (p *Error) Error() string {
	return fmt.Sprintf("status %d: %s", p.Status, p.Title)
}

func New(t string, status int, title string) *Error {
	return &Error{
		Type:   t,
		Status: status,
		Title:  title,
		Detail: "",
	}
}

// WithDetail returns new error that identical to existing, but contains
// additional detail.
//
//nolint:varnamelen
func (p *Error) WithDetail(d string) *Error {
	newErr := &Error{
		Type:   p.Type,
		Status: p.Status,
		Title:  p.Title,
		Detail: d,
	}

	if p.Detail != "" {
		newErr.Detail = strings.Join([]string{p.Detail, d}, "; ")
	}

	return newErr
}
