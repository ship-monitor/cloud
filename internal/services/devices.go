package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"charm.land/log/v2"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/pkg/names"
)

type StatesRepository interface {
	GetStates(
		ctx context.Context,
		deviceID, state string,
		historyLength int,
	) ([]domain.StateRecord, error)
}

type DeviceRepository interface {
	GetDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
	) (*domain.Device, error)
	ConnectDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
		userID uuid.UUID,
		name string,
	) (*domain.Device, error)
	RenameDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
		name string,
	) error
}

type DevicesService struct {
	devices        DeviceRepository
	states         StatesRepository
	logger         *log.Logger
	topicPublisher *TopicPublisher
	spicedb        *authzed.Client
}

var _ handlers.DevicesService = (*DevicesService)(nil)

func NewDevices(
	states StatesRepository,
	devices DeviceRepository,
	logger *log.Logger,
	topicPublisher *TopicPublisher,
	spicedb *authzed.Client,
) *DevicesService {
	return &DevicesService{
		devices:        devices,
		states:         states,
		logger:         logger,
		topicPublisher: topicPublisher,
		spicedb:        spicedb,
	}
}

var (
	ErrForbidden = fmt.Errorf(
		"%w: this action forbidden",
		domain.ErrForbidden,
	)
	ErrInvalidHistoryLength = errors.New("invalid history length specified")
)

func (d *DevicesService) ConnectDevice(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	password, name string,
) error {
	device, err := d.devices.GetDevice(ctx, deviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return handlers.ErrDeviceNotFound
	case err != nil:
		return fmt.Errorf("get device: %w", err)
	case device.OwnerID != nil:
		return fmt.Errorf(
			"connect device: %w",
			domain.ErrDeviceAlreadyConnected,
		)
	case !device.CheckPassword(password):
		return handlers.ErrInvalidDevicePassword
	}

	// TODO: return error if name is empty
	if name == "" {
		name = names.Gen()
	}

	if err := d.addRelation(
		ctx,
		deviceID,
		userID,
		DeviceRelationOwner,
	); err != nil {
		return fmt.Errorf("add relation: %w", err)
	}

	_, err = d.devices.ConnectDevice(ctx, deviceID, userID, name)
	switch {
	case errors.Is(err, domain.ErrDeviceAlreadyConnected):
		return fmt.Errorf(
			"connect device: %w",
			domain.ErrDeviceAlreadyConnected,
		)
	case err != nil:
		return fmt.Errorf("connect device: %w", err)
	default:
		return nil
	}
}

func (d *DevicesService) GetStates(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	state string,
	historyLength int,
) ([]domain.StateRecord, error) {
	if err := d.checkPermissions(
		ctx,
		deviceID,
		userID,
		DevicePermissionViewState,
	); err != nil {
		return nil, fmt.Errorf("check permissions: %w", err)
	}

	if historyLength < 0 {
		return nil, ErrInvalidHistoryLength
	}

	states, err := d.states.GetStates(
		ctx,
		deviceID.String(),
		state,
		historyLength,
	)
	if err != nil {
		return nil, fmt.Errorf("get states: %w", err)
	}

	return states, nil
}

func (d *DevicesService) SendCommand(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	command string,
	args any,
) error {
	if err := d.checkPermissions(
		ctx,
		deviceID,
		userID,
		DevicePermissionSendCommand,
	); err != nil {
		return fmt.Errorf("check permissions: %w", err)
	}

	cmd := Command{
		Command: command,
		Args:    args,
	}

	d.logger.Info(
		"Sending command",
		"deviceID", deviceID,
		"command", command,
		"topic", getDeeviceCommandTopic(deviceID),
	)

	err := d.topicPublisher.PublishJSON(
		ctx,
		getDeeviceCommandTopic(deviceID),
		cmd,
	)
	if err != nil {
		return fmt.Errorf("publish command: %w", err)
	}

	return nil
}

func (d *DevicesService) RenameDevice(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID uuid.UUID,
	name string,
) error {
	if err := d.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionRename,
	); err != nil {
		return fmt.Errorf("check permissions: %w", err)
	}

	err := d.devices.RenameDevice(ctx, deviceID, name)
	if err != nil {
		return fmt.Errorf("rename device: %w", err)
	}

	return nil
}

// checkPermissions checks if the user has the specified permission for the
// device. If user has no permission returned error will contain
// [domain.ErrForbidden].
func (d *DevicesService) checkPermissions(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	permission DevicePermission,
) error {
	result, err := d.spicedb.CheckPermission(ctx, &v1.CheckPermissionRequest{
		Resource: &v1.ObjectReference{
			ObjectType: "user",
			ObjectId:   userID.String(),
		},
		Permission: string(permission),
		Subject: &v1.SubjectReference{
			Object: &v1.ObjectReference{
				ObjectType: "device",
				ObjectId:   deviceID.String(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if result.GetPermissionship() != v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		return fmt.Errorf(
			"permission %q: %w",
			string(permission),
			domain.ErrForbidden,
		)
	}

	return nil
}

func (d *DevicesService) addRelation(
	ctx context.Context,
	deviceID, userID uuid.UUID,
	relation DeviceRelation,
) error {
	_, err := d.spicedb.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: []*v1.RelationshipUpdate{
			{
				Operation: v1.RelationshipUpdate_OPERATION_CREATE,
				Relationship: &v1.Relationship{
					Resource: &v1.ObjectReference{
						ObjectType: DeviceObjectType,
						ObjectId:   deviceID.String(),
					},
					Relation: string(relation),
					Subject: &v1.SubjectReference{
						Object: &v1.ObjectReference{
							ObjectType: UserObjectType,
							ObjectId:   userID.String(),
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("add relation: %w", err)
	}

	return nil
}

const (
	DeviceObjectType = "device"
	UserObjectType   = "user"
)

type DevicePermission string

const (
	DevicePermissionView        = DevicePermission("view")
	DevicePermissionViewState   = DevicePermission("view_state")
	DevicePermissionSendCommand = DevicePermission("send_command")
	DevicePermissionDisconnect  = DevicePermission("disconnect")
	DevicePermissionRename      = DevicePermission("rename")
	DevicePermissionShare       = DevicePermission("share")
)

type DeviceRelation string

const (
	DeviceRelationOwner  = DeviceRelation("owner")
	DeviceRelationAdmin  = DeviceRelation("administrator")
	DeviceRelationViewer = DeviceRelation("viewer")
)

type Command struct {
	Command string `json:"command"`
	Args    any    `json:"args"`
}

func getDeeviceCommandTopic(deviceID uuid.UUID) string {
	return fmt.Sprintf("devices.%s.commands", deviceID.String())
}
