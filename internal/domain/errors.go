package domain

import "errors"

// ErrForbidden is returned when the user is not have the required permissions
// to perform the requested action.
var ErrForbidden = errors.New("forbidden")
