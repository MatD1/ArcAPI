package services_test

import (
	"testing"

	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAPIKey(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	service := services.NewAuthService(nil, nil, nil, nil, cfg)

	key, hash, err := service.GenerateAPIKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, key, hash)
	assert.Greater(t, len(key), 32) // Should be base64 encoded 32 bytes
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	service := services.NewAuthService(nil, nil, nil, nil, cfg)

	user, err := service.ValidateJWT("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, user)
}

// Note: More comprehensive integration tests would require setting up actual database/mocks
// These provide basic unit tests for core functionality
