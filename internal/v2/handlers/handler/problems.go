package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"github.com/ship-monitor/cloud/pkg/problem"
)

var (
	errEmailTaken = problem.New(
		"email-taken",
		http.StatusConflict,
		"Email already taken",
	)

	errRegisterUser = problem.New(
		"register-user",
		http.StatusInternalServerError,
		"Failed create new user",
	)

	errUnauthenticated = problem.New(
		"unauthenticated",
		http.StatusUnauthorized,
		"User is unauthenticated",
	)

	errInvalidCredentials = problem.New(
		"invalid-credentials",
		http.StatusUnauthorized,
		"Invalid user credentials",
	)
	errLogin = problem.New(
		"login",
		http.StatusInternalServerError,
		"Login failed",
	)
	errUnknownError = problem.New(
		"unknown-error",
		http.StatusInternalServerError,
		"Unknown error occurred",
	).WithDetail("Please contact support")
)

func (h *Handler) NewError(
	ctx context.Context,
	err error,
) *api.BadResponseStatusCode {
	if p, ok := errors.AsType[*problem.Error](err); ok {
		return toBadResponse(p)
	}

	return toBadResponse(
		errUnknownError.WithDetail("Error message: " + err.Error()),
	)
}

//nolint:varnamelen
func toBadResponse(p *problem.Error) *api.BadResponseStatusCode {
	resp := &api.BadResponseStatusCode{
		StatusCode: p.Status,
		Response: api.Problem{
			Type:    p.Type,
			Title:   p.Title,
			Status:  float64(p.Status),
			Details: api.OptString{Set: false, Value: ""},
		},
	}

	if p.Detail != "" {
		resp.Response.Details = api.NewOptString(p.Detail)
	}

	return resp
}
