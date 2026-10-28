package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ogen-go/ogen/middleware"
	"github.com/ship-monitor/cloud/internal/v2/gen/api"
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
	)
	if err != nil {
		return nil, fmt.Errorf("new api server: %w", err)
	}

	return server, nil
}

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
		problem := api.Problem{
			Type:    "unknown",
			Title:   "unknown error",
			Details: api.NewOptString(err.Error()),
			Status:  http.StatusInternalServerError,
		}

		w.Header().Add("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)

		err = json.NewEncoder(w).Encode(&problem)
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
