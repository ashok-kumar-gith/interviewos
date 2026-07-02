package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository abstracts persistence for the auth domain so the service can be
// unit-tested against a fake. The gorm implementation is gormRepository.
type Repository interface {
	// Users.
	CreateUser(ctx context.Context, u *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	SetRoleByEmail(ctx context.Context, email string, role Role) error

	// Refresh tokens.
	CreateRefreshToken(ctx context.Context, t *RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID, at time.Time) error
	RevokeRefreshTokenFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID, at time.Time) error

	// Password reset tokens.
	CreatePasswordResetToken(ctx context.Context, t *PasswordResetToken) error
	GetResetTokenByHash(ctx context.Context, hash string) (*PasswordResetToken, error)
	MarkResetTokenUsed(ctx context.Context, id uuid.UUID, at time.Time) error

	// OAuth accounts.
	UpsertOAuthAccount(ctx context.Context, a *OAuthAccount) error
	FindOAuthAccount(ctx context.Context, provider Provider, providerUserID string) (*OAuthAccount, error)
}

// gormRepository is the GORM-backed Repository implementation.
type gormRepository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) CreateUser(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *gormRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	// gorm.DeletedAt on the model auto-filters deleted_at IS NULL.
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Update("last_login_at", at).Error
}

func (r *gormRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Update("password_hash", passwordHash).Error
}

// SetRoleByEmail promotes/demotes an account by email (used by the operational
// grant-admin flow). Returns ErrUserNotFound when no active account matches.
func (r *gormRepository) SetRoleByEmail(ctx context.Context, email string, role Role) error {
	res := r.db.WithContext(ctx).Model(&User{}).
		Where("lower(email) = lower(?)", normalizeEmail(email)).
		Update("role", role)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *gormRepository) CreateRefreshToken(ctx context.Context, t *RefreshToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *gormRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var t RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRefreshInvalid
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *gormRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID, at time.Time) error {
	updates := map[string]any{"revoked_at": at}
	if replacedBy != nil {
		updates["replaced_by"] = *replacedBy
	}
	return r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(updates).Error
}

func (r *gormRepository) RevokeRefreshTokenFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", at).Error
}

func (r *gormRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at).Error
}

func (r *gormRepository) CreatePasswordResetToken(ctx context.Context, t *PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *gormRepository) GetResetTokenByHash(ctx context.Context, hash string) (*PasswordResetToken, error) {
	var t PasswordResetToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrResetInvalid
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *gormRepository) MarkResetTokenUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&PasswordResetToken{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", at).Error
}

func (r *gormRepository) UpsertOAuthAccount(ctx context.Context, a *OAuthAccount) error {
	// Upsert on the (provider, provider_user_id) unique index.
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "provider_user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"email", "access_token", "refresh_token", "expires_at", "raw_profile", "updated_at",
		}),
	}).Create(a).Error
}

func (r *gormRepository) FindOAuthAccount(ctx context.Context, provider Provider, providerUserID string) (*OAuthAccount, error) {
	var a OAuthAccount
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
