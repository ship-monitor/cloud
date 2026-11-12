package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"github.com/ship-monitor/cloud/pkg/problem"
	"go.uber.org/zap"
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
	errRefreshToken = problem.New(
		"refresh-token",
		http.StatusUnauthorized,
		"Failed refresh token",
	)

	errDeviceNotFound = problem.New(
		"device-not-found",
		http.StatusNotFound,
		"Device not found",
	)

	errAlreadyConnected = problem.New(
		"device-already-connected",
		http.StatusConflict,
		"Device already connected to another user",
	)

	errInvalidPagination = problem.New(
		"invalid-pagination",
		http.StatusBadRequest,
		"Invalid pagination parameters",
	).WithDetail("Page must be a non-negative integer, " +
		"pageSize must be an integer between 1 and 50")

	errInvalidLimit = problem.New(
		"invalid-limit",
		http.StatusBadRequest,
		"Invalid limit parameter",
	).WithDetail("Limit must be a non-negative integer")

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
	h.logger.Error("Error occurred", zap.Error(err))

	if p, ok := errors.AsType[*problem.Error](err); ok {
		return toBadResponse(p)
	}

	return toBadResponse(
		errUnknownError.WithDetail("Error message: " + err.Error()),
	)
}

func toBadResponse(p *problem.Error) *api.BadResponseStatusCode {
	resp := &api.BadResponseStatusCode{
		StatusCode: p.Status,
		Response:   toApiProblem(p),
	}

	return resp
}

func toApiProblem(problem *problem.Error) api.Problem {
	apiP := api.Problem{
		Type:   problem.Type,
		Title:  problem.Title,
		Status: problem.Status,
		Detail: api.OptString{Set: false, Value: ""},
	}
	if problem.Detail != "" {
		apiP.Detail = api.NewOptString(problem.Detail)
	}

	return apiP
}
