package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/apolinario0x21/students-api/internal/models"
)

// GormUserRepository implementa o acesso a dados de usuários e refresh tokens.
type GormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormUserRepository) FindUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user := models.User{}
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) FindUserByID(ctx context.Context, id uint) (*models.User, error) {
	user := models.User{}
	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) ExistsUserWithUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (r *GormUserRepository) SaveRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindRefreshToken busca um refresh token pelo hash. Só retorna tokens válidos
// (não revogados); expiração é checada pela camada de serviço.
func (r *GormUserRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token := models.RefreshToken{}
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked = ?", tokenHash, false).
		First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrRefreshTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken marca um refresh token como revogado (idempotente).
func (r *GormUserRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked", true).Error
}

// PurgeExpiredRefreshTokens remove de vez (hard delete) os refresh tokens
// expirados ou revogados, devolvendo quantos foram apagados. Usa Unscoped porque
// gorm.Model faria apenas soft-delete.
func (r *GormUserRepository) PurgeExpiredRefreshTokens(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Unscoped().
		Where("expires_at < ? OR revoked = ?", now, true).
		Delete(&models.RefreshToken{})
	return res.RowsAffected, res.Error
}
