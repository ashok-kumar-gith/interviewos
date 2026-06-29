package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Provider enumerates the auth_provider Postgres enum values.
type Provider string

const (
	ProviderGoogle Provider = "google"
	ProviderGitHub Provider = "github"
	ProviderEmail  Provider = "email"
)

// Role enumerates the user_role Postgres enum values.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// Status enumerates the account_status Postgres enum values.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

// User is the canonical account record (table: users).
type User struct {
	ID              uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Email           string         `gorm:"column:email;type:text;not null"`
	EmailVerifiedAt *time.Time     `gorm:"column:email_verified_at"`
	PasswordHash    *string        `gorm:"column:password_hash"`
	FullName        *string        `gorm:"column:full_name"`
	AvatarURL       *string        `gorm:"column:avatar_url"`
	Role            Role           `gorm:"column:role;type:user_role;not null;default:user"`
	Status          Status         `gorm:"column:status;type:account_status;not null;default:active"`
	LastLoginAt     *time.Time     `gorm:"column:last_login_at"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (User) TableName() string { return "users" }

// EmailVerified is the API-facing derived boolean (no column).
func (u *User) EmailVerified() bool { return u.EmailVerifiedAt != nil }

// OAuthAccount is a linked external identity (table: oauth_accounts).
type OAuthAccount struct {
	ID             uuid.UUID      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`
	Provider       Provider       `gorm:"column:provider;type:auth_provider;not null"`
	ProviderUserID string         `gorm:"column:provider_user_id;not null"`
	Email          *string        `gorm:"column:email;type:text"`
	AccessToken    *string        `gorm:"column:access_token"`
	RefreshToken   *string        `gorm:"column:refresh_token"`
	ExpiresAt      *time.Time     `gorm:"column:expires_at"`
	RawProfile     []byte         `gorm:"column:raw_profile;type:jsonb"`
	CreatedAt      time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;not null;default:now()"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName pins the table name for GORM.
func (OAuthAccount) TableName() string { return "oauth_accounts" }

// RefreshToken is a rotating, hashed refresh-token record (table: refresh_tokens).
type RefreshToken struct {
	ID         uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID  `gorm:"column:user_id;type:uuid;not null"`
	TokenHash  string     `gorm:"column:token_hash;not null"`
	FamilyID   uuid.UUID  `gorm:"column:family_id;type:uuid;not null"`
	UserAgent  *string    `gorm:"column:user_agent"`
	IPAddress  *string    `gorm:"column:ip_address;type:inet"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	ReplacedBy *uuid.UUID `gorm:"column:replaced_by;type:uuid"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (RefreshToken) TableName() string { return "refresh_tokens" }

// IsActive reports whether the token is neither revoked nor expired.
func (t *RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

// PasswordResetToken is a single-use, expiring reset record
// (table: password_reset_tokens).
type PasswordResetToken struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID  `gorm:"column:user_id;type:uuid;not null"`
	TokenHash string     `gorm:"column:token_hash;not null"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null;default:now()"`
}

// TableName pins the table name for GORM.
func (PasswordResetToken) TableName() string { return "password_reset_tokens" }
