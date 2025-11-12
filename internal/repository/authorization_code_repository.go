package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/mat/arcapi/internal/models"
	"gorm.io/gorm"
)

type AuthorizationCodeRepository struct { db *gorm.DB }

func NewAuthorizationCodeRepository(db *DB) *AuthorizationCodeRepository { return &AuthorizationCodeRepository{db: db.DB} }

func (r *AuthorizationCodeRepository) Create(userID uint, plainCode, codeChallenge, method string, ttl time.Duration) error {
	hash := sha256.Sum256([]byte(plainCode))
	code := models.AuthorizationCode{
		UserID:             userID,
		CodeHash:           hex.EncodeToString(hash[:]),
		CodeChallenge:      codeChallenge,
		CodeChallengeMethod: method,
		ExpiresAt:          time.Now().Add(ttl),
	}
	return r.db.Create(&code).Error
}

func (r *AuthorizationCodeRepository) FindByPlain(plainCode string) (*models.AuthorizationCode, error) {
	hash := sha256.Sum256([]byte(plainCode))
	codeHash := hex.EncodeToString(hash[:])
	var code models.AuthorizationCode
	if err := r.db.Where("code_hash = ?", codeHash).First(&code).Error; err != nil { return nil, err }
	return &code, nil
}

func (r *AuthorizationCodeRepository) Consume(code *models.AuthorizationCode) error {
	if code.ConsumedAt != nil { return nil }
	now := time.Now()
	return r.db.Model(code).Update("consumed_at", &now).Error
}
