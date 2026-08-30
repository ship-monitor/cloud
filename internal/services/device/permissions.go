package device

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/google/uuid"
	"github.com/ship-monitor/cloud/internal/domain"
)

func (s *Service) lookupAccessibleDeviceIDs(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.DeviceID, error) {
	stream, err := s.spicedb.LookupResources(
		ctx,
		&v1.LookupResourcesRequest{ //nolint:exhaustruct_v5
			Consistency: &v1.Consistency{
				Requirement: &v1.Consistency_FullyConsistent{
					FullyConsistent: true,
				},
			},
			ResourceObjectType: DeviceObjectType,
			Permission:         string(DevicePermissionView),
			Subject: &v1.SubjectReference{ //nolint:exhaustruct_v5
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
func (s *Service) checkPermissions(
	ctx context.Context,
	deviceID domain.DeviceID, userID uuid.UUID,
	permission DevicePermission,
) error {
	result, err := s.spicedb.CheckPermission(
		ctx,
		&v1.CheckPermissionRequest{ //nolint:exhaustruct_v5
			Resource: &v1.ObjectReference{
				ObjectType: DeviceObjectType,
				ObjectId:   deviceID.String(),
			},
			Permission: string(permission),
			Subject: &v1.SubjectReference{ //nolint:exhaustruct_v5
				Object: &v1.ObjectReference{
					ObjectType: UserObjectType,
					ObjectId:   userID.String(),
				},
			},
		},
	)
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

func (s *Service) addRelation(
	ctx context.Context,
	deviceID domain.DeviceID, userID uuid.UUID,
	relation DeviceRelation,
) error {
	_, err := s.spicedb.WriteRelationships(
		ctx,
		&v1.WriteRelationshipsRequest{ //nolint:exhaustruct_v5
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
		},
	)
	if err != nil {
		return fmt.Errorf("add relation: %w", err)
	}

	return nil
}

// setAccessRelation atomically replaces a viewer/administrator relation.
// The owner relation is intentionally left unchanged.
func (s *Service) setAccessRelation(
	ctx context.Context,
	deviceID domain.DeviceID,
	userID uuid.UUID,
	relation DeviceRelation,
) error {
	oppositeRelation := DeviceRelationViewer
	if relation == DeviceRelationViewer {
		oppositeRelation = DeviceRelationAdmin
	}

	_, err := s.spicedb.WriteRelationships(
		ctx,
		&v1.WriteRelationshipsRequest{ //nolint:exhaustruct_v5
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

func (s *Service) deleteAccessRelations(
	ctx context.Context,
	deviceID domain.DeviceID,
	userID uuid.UUID,
) error {
	_, err := s.spicedb.WriteRelationships(
		ctx,
		&v1.WriteRelationshipsRequest{ //nolint:exhaustruct_v5
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

func (s *Service) readDeviceAccess(
	ctx context.Context,
	deviceID domain.DeviceID,
) ([]domain.DeviceAccess, error) {
	stream, err := s.spicedb.ReadRelationships(
		ctx,
		&v1.ReadRelationshipsRequest{ //nolint:exhaustruct_v5
			Consistency: &v1.Consistency{
				Requirement: &v1.Consistency_FullyConsistent{
					FullyConsistent: true,
				},
			},
			RelationshipFilter: &v1.RelationshipFilter{ //nolint:exhaustruct_v5
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
	return &v1.Relationship{ //nolint:exhaustruct_v5
		Resource: &v1.ObjectReference{
			ObjectType: DeviceObjectType,
			ObjectId:   deviceID.String(),
		},
		Relation: string(relation),
		Subject: &v1.SubjectReference{ //nolint:exhaustruct_v5
			Object: &v1.ObjectReference{
				ObjectType: UserObjectType,
				ObjectId:   userID.String(),
			},
		},
	}
}

// clearRelations deletes all permissions for spicified device ID. Suitable
// for disconnecting device.
func (s *Service) clearRelations(
	ctx context.Context,
	deviceID domain.DeviceID,
) error {
	_, err := s.spicedb.DeleteRelationships(ctx,
		&v1.DeleteRelationshipsRequest{ //nolint:exhaustruct_v5
			RelationshipFilter: &v1.RelationshipFilter{ //nolint:exhaustruct_v5
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
