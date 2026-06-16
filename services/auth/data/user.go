package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

var ErrEmailAlreadyTaken = errors.New("email already taken")

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

// NewUser creates new user in database.
// Returns [ErrEmailAlreadyTaken].
func NewUser(name, email, password string) (*User, error) {
	if name == "" || email == "" || password == "" {
		return nil, fmt.Errorf("name, email and password are required")
	}

	if taken, err := checkEmailTaken(email); err != nil {
		return nil, fmt.Errorf("failed check email availability: %w", err)
	} else if taken {
		return nil, fmt.Errorf("new user: %w", ErrEmailAlreadyTaken)
	}

	user := User{
		ID:            uuid.New(),
		Name:          name,
		Email:         normalizeEmail(email),
		PasswordHash:  hashPassword(password),
		EmailVerified: false,
		Blocked:       false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if _, err := db.DB.NewInsert().Model(&user).Exec(context.TODO()); err != nil {
		return nil, fmt.Errorf("new user: %w", err)
	}

	return &user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(email)
}

func hashPassword(password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("failed hash password: %w", err))
	}

	return hash
}

func (u *User) ComparePassword(password string) bool {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)) == nil
}

func GetUser(id uuid.UUID) (*User, error) {
	var user User
	err := db.DB.NewSelect().Model(&user).Where("id = ?", id).Scan(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}

func GetUserByEmail(email string) (*User, error) {
	email = normalizeEmail(email)
	var user User
	err := db.DB.NewSelect().Model(&user).Where("email = ?", email).Scan(context.TODO())
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)) == nil
}

func checkEmailTaken(email string) (bool, error) {
	taken, err := db.DB.NewSelect().
		Model((*User)(nil)).
		Column("email").
		Where("email = ?", normalizeEmail(email)).
		Exists(context.TODO())
	if err != nil {
		return false, err
	}

	return taken, nil
}

func (u *User) SetPassword(password string) error {
	u.UpdatedAt = time.Now()
	u.PasswordHash = hashPassword(password)

	_, err := db.DB.NewUpdate().
		Model(u).
		Column("password_hash", "updated_at").
		WherePK().
		Exec(context.TODO())

	return err
}

func (u *User) SetEmail(email string) error {
	u.UpdatedAt = time.Now()
	u.Email = normalizeEmail(email)
	u.EmailVerified = false

	_, err := db.DB.NewUpdate().
		Model(u).
		Column("email", "email_verified", "updated_at").
		WherePK().
		Exec(context.TODO())

	return err
}

func (u *User) Block() error {
	u.UpdatedAt = time.Now()
	u.Blocked = true
	_, err := db.DB.NewUpdate().
		Model(u).
		Column("blocked", "updated_at").
		WherePK().
		Exec(context.TODO())

	return err
}

const PageLimit = 20

func GetUsersList(page int) ([]User, error) {
	if page < 0 {
		return nil, fmt.Errorf("invalid page value '%d'", page)
	}
	var users []User
	err := db.DB.NewSelect().
		Model(&users).
		Offset(page * PageLimit).
		Limit(PageLimit).
		Scan(context.TODO())
	if err != nil {
		return nil, err
	}

	return users, nil
}
