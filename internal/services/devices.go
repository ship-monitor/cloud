package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"sort"

	"charm.land/log/v2"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/handlers"
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
	GetDevicesByIDs(
		ctx context.Context,
		deviceIDs []domain.DeviceID,
	) ([]domain.Device, error)
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
	DisconnectDevice(
		ctx context.Context,
		deviceID domain.DeviceID,
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
	ErrInvalidHistoryLength = errors.New(
		"invalid history length specified",
	)
	errAccessRelationshipMissingSubject = errors.New(
		"device access relationship has no subject",
	)
)

func (d *DevicesService) ConnectDevice(
	ctx context.Context,
	applicant *domain.Principal,
	in handlers.ConnectDeviceIn,
) error {
	if err := validator.New().Struct(&in); err != nil {
		return fmt.Errorf("invalid input data: %w", err)
	}

	device, err := d.devices.GetDevice(ctx, in.DeviceID)
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
	case !device.CheckPassword(in.Password):
		return handlers.ErrInvalidDevicePassword
	}

	if err := d.addRelation(
		ctx,
		in.DeviceID,
		applicant.UserID,
		DeviceRelationOwner,
	); err != nil {
		return fmt.Errorf("add relation: %w", err)
	}

	_, err = d.devices.ConnectDevice(
		ctx,
		in.DeviceID,
		applicant.UserID,
		in.Name,
	)
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

func (d *DevicesService) GetDevice(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID domain.DeviceID,
) (*domain.Device, error) {
	if err := d.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionView,
	); err != nil {
		return nil, fmt.Errorf("check permission: %w", err)
	}

	device, err := d.devices.GetDevice(ctx, deviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("get device: %w", handlers.ErrDeviceNotFound)
	case err != nil:
		return nil, fmt.Errorf("get device: %w", err)
	default:
		return device, nil
	}
}

func (d *DevicesService) GetDevices(
	ctx context.Context,
	applicant *domain.Principal,
) ([]domain.Device, error) {
	deviceIDs, err := d.lookupAccessibleDeviceIDs(ctx, applicant.UserID)
	if err != nil {
		return nil, fmt.Errorf("lookup accessible devices: %w", err)
	}

	devices, err := d.devices.GetDevicesByIDs(ctx, deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("get devices by IDs: %w", err)
	}

	return devices, nil
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

func (d *DevicesService) SetDeviceAccess(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID uuid.UUID,
	access domain.DeviceAccess,
) error {
	relation, err := deviceRelationForAccessRole(access.Role)
	if err != nil {
		return fmt.Errorf("resolve device access role: %w", err)
	}

	if err := d.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionShare,
	); err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if err := d.setAccessRelation(
		ctx,
		deviceID,
		access.UserID,
		relation,
	); err != nil {
		return fmt.Errorf("set access relation: %w", err)
	}

	return nil
}

func (d *DevicesService) DeleteDeviceAccess(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID, userID uuid.UUID,
) error {
	if err := d.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionShare,
	); err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if err := d.deleteAccessRelations(ctx, deviceID, userID); err != nil {
		return fmt.Errorf("delete access relations: %w", err)
	}

	return nil
}

func (d *DevicesService) GetDeviceAccess(
	ctx context.Context,
	applicant *domain.Principal,
	deviceID uuid.UUID,
) ([]domain.DeviceAccess, error) {
	if err := d.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionShare,
	); err != nil {
		return nil, fmt.Errorf("check permission: %w", err)
	}

	access, err := d.readDeviceAccess(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("read device access: %w", err)
	}

	return access, nil
}

func (d *DevicesService) DisconnectDevice(
	ctx context.Context,
	deviceID domain.DeviceID,
	applicant *domain.Principal,
) error {
	if err := d.checkPermissions(
		ctx,
		deviceID,
		applicant.UserID,
		DevicePermissionDisconnect,
	); err != nil {
		return fmt.Errorf("check permission: %w", err)
	}

	if err := d.devices.DisconnectDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("repo disconnect device: %w", err)
	}

	if err := d.clearRelations(ctx, deviceID); err != nil {
		return fmt.Errorf("clear permissions: %w", err)
	}

	return nil
}

func (d *DevicesService) lookupAccessibleDeviceIDs(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.DeviceID, error) {
	stream, err := d.spicedb.LookupResources(
		ctx,
		&v1.LookupResourcesRequest{
			Consistency: &v1.Consistency{
				Requirement: &v1.Consistency_FullyConsistent{
					FullyConsistent: true,
				},
			},
			ResourceObjectType: DeviceObjectType,
			Permission:         string(DevicePermissionView),
			Subject: &v1.SubjectReference{
				Object: &v1.ObjectReference{
					ObjectType: UserObjectType,
					ObjectId:   userID.String(),
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("lookup resources: %w", err)
	}

	deviceIDsByID := make(map[domain.DeviceID]struct{})

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("receive resource: %w", err)
		}

		if response.GetPermissionship() !=
			v1.LookupPermissionship_LOOKUP_PERMISSIONSHIP_HAS_PERMISSION {
			continue
		}

		deviceID, err := uuid.Parse(response.GetResourceObjectId())
		if err != nil {
			return nil, fmt.Errorf("parse accessible device ID: %w", err)
		}

		deviceIDsByID[deviceID] = struct{}{}
	}

	deviceIDs := make([]domain.DeviceID, 0, len(deviceIDsByID))
	for deviceID := range deviceIDsByID {
		deviceIDs = append(deviceIDs, deviceID)
	}

	sort.Slice(deviceIDs, func(i, j int) bool {
		return deviceIDs[i].String() < deviceIDs[j].String()
	})

	return deviceIDs, nil
}

// checkPermissions checks if the user has the specified permission for the
// device. If user has no permission returned error will contain
// [domain.ErrForbidden].
func (d *DevicesService) checkPermissions(
	ctx context.Context,
	deviceID domain.DeviceID, userID uuid.UUID,
	permission DevicePermission,
) error {
	result, err := d.spicedb.CheckPermission(ctx, &v1.CheckPermissionRequest{
		Resource: &v1.ObjectReference{
			ObjectType: DeviceObjectType,
			ObjectId:   deviceID.String(),
		},
		Permission: string(permission),
		Subject: &v1.SubjectReference{
			Object: &v1.ObjectReference{
				ObjectType: UserObjectType,
				ObjectId:   userID.String(),
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
	deviceID domain.DeviceID, userID uuid.UUID,
	relation DeviceRelation,
) error {
	_, err := d.spicedb.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: []*v1.RelationshipUpdate{
			{
				Operation: v1.RelationshipUpdate_OPERATION_TOUCH,
				Relationship: deviceRelationship(
					deviceID,
					userID,
					relation,
				),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("add relation: %w", err)
	}

	return nil
}

// setAccessRelation atomically replaces a viewer/administrator relation.
// The owner relation is intentionally left unchanged.
func (d *DevicesService) setAccessRelation(
	ctx context.Context,
	deviceID domain.DeviceID,
	userID uuid.UUID,
	relation DeviceRelation,
) error {
	oppositeRelation := DeviceRelationViewer
	if relation == DeviceRelationViewer {
		oppositeRelation = DeviceRelationAdmin
	}

	_, err := d.spicedb.WriteRelationships(
		ctx,
		&v1.WriteRelationshipsRequest{
			Updates: []*v1.RelationshipUpdate{
				{
					Operation: v1.RelationshipUpdate_OPERATION_DELETE,
					Relationship: deviceRelationship(
						deviceID,
						userID,
						oppositeRelation,
					),
				},
				{
					Operation: v1.RelationshipUpdate_OPERATION_TOUCH,
					Relationship: deviceRelationship(
						deviceID,
						userID,
						relation,
					),
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("write access relation: %w", err)
	}

	return nil
}

func (d *DevicesService) deleteAccessRelations(
	ctx context.Context,
	deviceID domain.DeviceID,
	userID uuid.UUID,
) error {
	_, err := d.spicedb.WriteRelationships(
		ctx,
		&v1.WriteRelationshipsRequest{
			Updates: []*v1.RelationshipUpdate{
				{
					Operation: v1.RelationshipUpdate_OPERATION_DELETE,
					Relationship: deviceRelationship(
						deviceID,
						userID,
						DeviceRelationAdmin,
					),
				},
				{
					Operation: v1.RelationshipUpdate_OPERATION_DELETE,
					Relationship: deviceRelationship(
						deviceID,
						userID,
						DeviceRelationViewer,
					),
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("write relationship deletions: %w", err)
	}

	return nil
}

func (d *DevicesService) readDeviceAccess(
	ctx context.Context,
	deviceID domain.DeviceID,
) ([]domain.DeviceAccess, error) {
	stream, err := d.spicedb.ReadRelationships(
		ctx,
		&v1.ReadRelationshipsRequest{
			Consistency: &v1.Consistency{
				Requirement: &v1.Consistency_FullyConsistent{
					FullyConsistent: true,
				},
			},
			RelationshipFilter: &v1.RelationshipFilter{
				ResourceType:       DeviceObjectType,
				OptionalResourceId: deviceID.String(),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read relationships: %w", err)
	}

	rolesByUser := make(map[uuid.UUID]domain.DeviceAccessRole)

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("receive relationship: %w", err)
		}

		relationship := response.GetRelationship()
		if relationship == nil {
			continue
		}

		role, ok := deviceAccessRoleForRelation(relationship.GetRelation())
		if !ok {
			continue
		}

		subject := relationship.GetSubject()
		if subject == nil || subject.GetObject() == nil {
			return nil, fmt.Errorf(
				"read relationship: %w",
				errAccessRelationshipMissingSubject,
			)
		}

		if subject.GetObject().GetObjectType() != UserObjectType {
			continue
		}

		userID, err := uuid.Parse(subject.GetObject().GetObjectId())
		if err != nil {
			return nil, fmt.Errorf("parse access user ID: %w", err)
		}

		currentRole, exists := rolesByUser[userID]
		if !exists || currentRole == domain.DeviceAccessRoleViewer {
			rolesByUser[userID] = role
		}
	}

	access := make([]domain.DeviceAccess, 0, len(rolesByUser))
	for userID, role := range rolesByUser {
		access = append(access, domain.DeviceAccess{
			UserID: userID,
			Role:   role,
		})
	}

	sort.Slice(access, func(i, j int) bool {
		return access[i].UserID.String() < access[j].UserID.String()
	})

	return access, nil
}

func deviceRelationForAccessRole(
	role domain.DeviceAccessRole,
) (DeviceRelation, error) {
	switch role {
	case domain.DeviceAccessRoleAdmin:
		return DeviceRelationAdmin, nil
	case domain.DeviceAccessRoleViewer:
		return DeviceRelationViewer, nil
	default:
		return "", fmt.Errorf(
			"%w: %q",
			domain.ErrInvalidDeviceAccessRole,
			role,
		)
	}
}

func deviceAccessRoleForRelation(
	relation string,
) (domain.DeviceAccessRole, bool) {
	switch DeviceRelation(relation) {
	case DeviceRelationAdmin:
		return domain.DeviceAccessRoleAdmin, true
	case DeviceRelationViewer:
		return domain.DeviceAccessRoleViewer, true
	case DeviceRelationOwner:
		return "", false
	default:
		return "", false
	}
}

func deviceRelationship(
	deviceID domain.DeviceID,
	userID uuid.UUID,
	relation DeviceRelation,
) *v1.Relationship {
	return &v1.Relationship{
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
	}
}

// clearRelations deletes all permissions for spicified device ID. Suitable
// for disconnecting device.
func (d *DevicesService) clearRelations(
	ctx context.Context,
	deviceID domain.DeviceID,
) error {
	_, err := d.spicedb.DeleteRelationships(ctx,
		&v1.DeleteRelationshipsRequest{
			RelationshipFilter: &v1.RelationshipFilter{
				ResourceType:       DeviceObjectType,
				OptionalResourceId: deviceID.String(),
			},
		})
	if err != nil {
		return fmt.Errorf("delete relationships: %w", err)
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
