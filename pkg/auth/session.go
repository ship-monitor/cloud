package auth

import (
	"fmt"

	"charm.land/log/v2"
	api "github.com/authzed/authzed-go/proto/authzed/api/v1"
	authzed "github.com/authzed/authzed-go/v1"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Session struct {
	spiceDB *authzed.Client
	UserID  uuid.UUID `json:"userId"`
	Email   string    `json:"email"`
	c       *gin.Context
}

// SpiceDB returns client.
//
// Warning: Do not rely on this method; it will be removed in a future release. Not deprecated, but
// should not be used directly.
func (s *Session) SpiceDB() *authzed.Client {
	log.Warn("Usage of Session.SpiceDB()")

	return s.spiceDB
}

const (
	UserObjectType = "user"
)

func (s *Session) CheckPermission(resource, resourceID, permission string) (bool, error) {
	response, err := s.spiceDB.CheckPermission(s.c.Request.Context(), &api.CheckPermissionRequest{
		Resource: &api.ObjectReference{
			ObjectType: resource,
			ObjectId:   resourceID,
		},
		Subject: &api.SubjectReference{
			Object: &api.ObjectReference{
				ObjectType: UserObjectType,
				ObjectId:   s.UserID.String(),
			},
		},
		Permission: permission,
	})
	if err != nil {
		log.Error("Failed to check permission", "error", err)

		return false, fmt.Errorf("check permission: %w", err)
	}

	hasPermission := response.GetPermissionship() == api.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION

	return hasPermission, nil
}
