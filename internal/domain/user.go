package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	*bun.BaseModel `bun:"table:users"`

	ID            uuid.UUID `bun:",pk,type:varchar" json:"id"`
	Name          string    `bun:",notnull" json:"name"`
	Email         string    `bun:",unique" json:"email"`
	EmailVerified bool      `bun:",notnull,default:false" json:"emailVerified"`
	PasswordHash  []byte    `bun:",notnull" json:"-"`
	Blocked       bool      `bun:",notnull,default:false" json:"blocked"`
	CreatedAt     time.Time `bun:",nullzero,notnull,type:varchar" json:"createdAt"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,type:varchar" json:"updatedAt"`
}

func NormalizeEmail(email string) string {
	return strings.ToLower(email)
}

type RegisterUserData struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=5"`
}
