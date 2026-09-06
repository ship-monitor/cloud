package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"charm.land/log/v2"
	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
	connection "github.com/ship-monitor/protocol/go/connection/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	connection.UnimplementedConnectionServiceServer

	logger      *log.Logger
	service     Service
	authService HubAuthService
}

var _ connection.ConnectionServiceServer = (*Server)(nil)

func New(
	logger *log.Logger,
	service Service,
	authService HubAuthService,
) *Server {
	return &Server{
		UnimplementedConnectionServiceServer: connection.UnimplementedConnectionServiceServer{},
		logger:                               logger,
		service:                              service,
		authService:                          authService,
	}
}

var errNoMetadata = errors.New("no metadata in coming context")

// ConnectDeviceToHub implements [v1.ConnectionServiceServer].
func (s *Server) ConnectDeviceToHub(
	ctx context.Context,
	req *connection.ConnectDeviceRequest,
) (*connection.ConnectDeviceResponse, error) {
	claims, err := s.authHub(ctx)
	if err != nil {
		s.logger.Error("Failed auhenticate HUB", "error", err)

		return &connection.ConnectDeviceResponse{
			Approved: false,
			Error:    connection.ConnectionError_CONNECTION_ERROR_HUB_NOT_AUTHENTICATED.Enum(),
		}, nil
	}

	err = s.service.ConnectDevice(ctx, domain.ConnectDeviceRequest{
		DeviceID:  uuid.MustParse(req.GetDeviceId()),
		DeviceKey: req.GetDeviceKey(),
		HubID:     claims.HubID,
	})
	if err != nil {
		s.logger.Error("Failed to connect device to HUB",
			"error", err,
			"hubID", claims.HubID,
		)

		return &connection.ConnectDeviceResponse{
			Approved: false,
			Error:    connection.ConnectionError_CONNECTION_ERROR_INTERNAL.Enum(),
		}, nil
	}

	return &connection.ConnectDeviceResponse{
		Approved: true,
		Error:    nil,
	}, nil
}

// Connect implements [v1.ConnectionServiceServer].
func (s *Server) Connect(
	streaming grpc.BidiStreamingServer[connection.HubMessage, connection.ServerMessage],
) error {
	var group errgroup.Group

	group.Go(func() error { return s.handleHubMessages(streaming) })
	group.Go(func() error { return s.handleServerMessages(streaming) })

	err := group.Wait()
	if err != nil {
		return fmt.Errorf("group wait: serving bidirectional messages: %w", err)
	}

	return nil
}

func (s *Server) AuthenticateHub(
	ctx context.Context,
	req *connection.AuthenticateHubRequest,
) (*connection.AuthenticateHubResponse, error) {
	data, err := s.authService.AuthenticateHub(
		ctx,
		uuid.MustParse(req.GetHubId()),
		req.GetModel(),
	)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"authenticate hub: %v", err,
		)
	}

	return &connection.AuthenticateHubResponse{
		Auth: &connection.AuthData{
			AccessToken:  data.AccessToken,
			RefreshToken: data.RefreshToken,
		},
	}, nil
}

func (s *Server) RefreshHubAuth(
	context.Context,
	*connection.RefreshHubAuthRequest,
) (*connection.RefreshHubAuthResponse, error) {
	return nil, status.Errorf(
		codes.Unimplemented,
		"method RefreshHubAuth not implemented",
	)
}

func (s *Server) handleHubMessages(
	streaming grpc.BidiStreamingServer[connection.HubMessage, connection.ServerMessage],
) error {
	for {
		_, err := streaming.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("receive next message: %w", err)
		}
	}
}

func (s *Server) handleServerMessages(
	streaming grpc.BidiStreamingServer[connection.HubMessage, connection.ServerMessage],
) error {
	return nil
}

type Principal struct {
	HubID uuid.UUID
}

func (s *Server) authHub(ctx context.Context) (*Principal, error) {
	_, exists := metadata.FromIncomingContext(ctx)
	if !exists {
		return nil, errNoMetadata
	}

	// TODO: write auth

	return nil, errors.New("not implemented")
}
