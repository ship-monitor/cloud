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

func (p *Error) WithDetail(d string) *Error {
	return &Error{
		Type:   p.Type,
		Status: p.Status,
		Title:  p.Title,
		Detail: strings.Join([]string{p.Detail, d}, "; "),
	}
}
