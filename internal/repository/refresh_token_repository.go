package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/mat/arcapi/internal/models"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct { db *gorm.DB }

func NewRefreshTokenRepository(db *DB) *RefreshTokenRepository { return &RefreshTokenRepository{db: db.DB} }

func (r *RefreshTokenRepository) Create(userID uint, plainToken string, expiry time.Time) error {
	hash := sha256.Sum256([]byte(plainToken))
	rt := models.RefreshToken{ UserID: userID, TokenHash: hex.EncodeToString(hash[:]), ExpiresAt: expiry }
	return r.db.Create(&rt).Error
}

func (r *RefreshTokenRepository) FindByPlain(plainToken string) (*models.RefreshToken, error) {
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hex.EncodeToString(hash[:])
	var rt models.RefreshToken
	if err := r.db.Where("token_hash = ?", tokenHash).First(&rt).Error; err != nil { return nil, err }
	return &rt, nil
}

func (r *RefreshTokenRepository) Revoke(rt *models.RefreshToken) error {
	now := time.Now()
	return r.db.Model(rt).Updates(map[string]interface{}{"revoked_at": &now}).Error
}

func (r *RefreshTokenRepository) Touch(rt *models.RefreshToken) error {
	now := time.Now()
	return r.db.Model(rt).Update("last_used_at", &now).Error
}
