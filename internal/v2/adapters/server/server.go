package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ogen-go/ogen/middleware"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
	"github.com/ship-monitor/cloud/pkg/problem"
	"go.uber.org/zap"
)

func New(
	logger *zap.Logger,
	handler api.Handler,
	security api.SecurityHandler,
) (*api.Server, error) {
	logger = logger.Named("HTTP")

	server, err := api.NewServer(
		handler,
		security,
		api.WithErrorHandler(ErrorHandler(logger)),
		api.WithMiddleware(Logging(logger)),
		api.WithDocs("/docs"),
	)
	if err != nil {
		return nil, fmt.Errorf("new api server: %w", err)
	}

	return server, nil
}

var errBadRequest = problem.New(
	"bad-request",
	http.StatusBadRequest,
	"Bad request",
)

//nolint:varnamelen
func ErrorHandler(logger *zap.Logger) api.ErrorHandler {
	logger = logger.Named("Error Handler")

	return func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
		err error,
	) {
		logger = logger.With(
			zap.String("method", r.Method),
			zap.String("url", r.RequestURI),
		)

		logger.Error("Error occurred while handling request",
			zap.Error(err),
		)

		problem := errBadRequest

		if p, ok := errors.AsType[*ogenerrors.DecodeParamsError](err); ok {
			problem = problem.WithDetail(
				fmt.Sprintf(
					"Decode params: %v", p.Err,
				),
			)
		}

		if p, ok := errors.AsType[*ogenerrors.DecodeParamError](err); ok {
			problem = problem.WithDetail(
				fmt.Sprintf("Decode param in %s: %v", p.In, p.Err),
			)
			problem.Title = "Bad request param"
		}

		if p, ok := errors.AsType[*ogenerrors.DecodeBodyError](err); ok {
			problem = problem.WithDetail(
				fmt.Sprintf("Decode body: %v", p.Err),
			)
			problem.Title = "Bad request body"
		}

		if p, ok := errors.AsType[*ogenerrors.DecodeRequestError](err); ok {
			problem = problem.WithDetail(
				fmt.Sprintf(
					"Decode request: %v", p.Err,
				),
			)
		}

		w.Header().Add("Content-Type", "application/problem+json")
		w.WriteHeader(problem.Status)

		err = json.NewEncoder(w).Encode(&toBadResponse(problem).Response)
		if err != nil {
			logger.Error(
				"Failed encode problem to response body",
				zap.Error(err),
			)

			return
		}
	}
}

func Logging(logger *zap.Logger) api.Middleware {
	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		logger.Info(
			fmt.Sprintf(
				"%s %s - %s",
				req.Raw.Method,
				req.Raw.URL,
				req.OperationID,
			),
		)

		return next(req)
	}
}

//nolint:varnamelen
func toBadResponse(p *problem.Error) *api.BadResponseStatusCode {
	resp := &api.BadResponseStatusCode{
		StatusCode: p.Status,
		Response: api.Problem{
			Type:   p.Type,
			Title:  p.Title,
			Status: float64(p.Status),
			Detail: api.OptString{Set: false, Value: ""},
		},
	}

	if p.Detail != "" {
		resp.Response.Detail = api.NewOptString(p.Detail)
	}

	return resp
}
