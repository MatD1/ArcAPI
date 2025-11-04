package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo     *repository.UserRepository
	apiKeyRepo   *repository.APIKeyRepository
	jwtTokenRepo *repository.JWTTokenRepository
	cacheService *CacheService
	cfg          *config.Config
	jwtSecret    []byte
}

func NewAuthService(
	userRepo *repository.UserRepository,
	apiKeyRepo *repository.APIKeyRepository,
	jwtTokenRepo *repository.JWTTokenRepository,
	cacheService *CacheService,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		apiKeyRepo:   apiKeyRepo,
		jwtTokenRepo: jwtTokenRepo,
		cacheService: cacheService,
		cfg:          cfg,
		jwtSecret:    []byte(cfg.JWTSecret),
	}
}

// GenerateAPIKey generates a new API key and returns both the plain key and hashed version
func (s *AuthService) GenerateAPIKey() (string, string, error) {
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return "", "", err
	}
	key := base64.URLEncoding.EncodeToString(keyBytes)

	hashed, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	return key, string(hashed), nil
}

// ValidateAPIKey validates an API key and returns the associated APIKey
// Note: API keys are tied to user accounts, so access control is checked via the user's CanAccessData
func (s *AuthService) ValidateAPIKey(apiKey string) (*models.APIKey, error) {
	// Check cache first (if available)
	if s.cacheService != nil {
		cacheKey := APIKeyCacheKey(apiKey)
		var cachedKey models.APIKey
		err := s.cacheService.GetJSON(cacheKey, &cachedKey)
		if err == nil && cachedKey.ID > 0 {
			if cachedKey.IsRevoked() {
				return nil, fmt.Errorf("API key is revoked")
			}
			// Always fetch fresh user data to check CanAccessData
			user, err := s.userRepo.FindByID(cachedKey.UserID)
			if err == nil {
				cachedKey.User = *user
			}
			// Update last used in background
			go s.apiKeyRepo.UpdateLastUsed(cachedKey.ID)
			return &cachedKey, nil
		}
	}

	// Find all active keys and check hash
	// Note: bcrypt includes salt, so we must check each key
	keys, err := s.apiKeyRepo.FindAllActive()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(apiKey))
		if err == nil {
			// Always fetch fresh user data to ensure CanAccessData is current
			user, err := s.userRepo.FindByID(key.UserID)
			if err == nil {
				key.User = *user
			}
			// Cache for 5 minutes (if available)
			if s.cacheService != nil {
				cacheKey := APIKeyCacheKey(apiKey)
				s.cacheService.SetJSON(cacheKey, key, 5*time.Minute)
			}
			// Update last used
			go s.apiKeyRepo.UpdateLastUsed(key.ID)
			return &key, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// GenerateJWT generates a JWT token for a user
func (s *AuthService) GenerateJWT(user *models.User) (string, error) {
	// Validate user
	if user == nil || user.ID == 0 {
		return "", fmt.Errorf("invalid user: user is nil or ID is 0")
	}

	// Validate JWT secret
	if len(s.jwtSecret) == 0 {
		return "", fmt.Errorf("JWT secret is not configured")
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour)

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	// Hash token for storage (use SHA-256 since JWT tokens can exceed bcrypt's 72-byte limit)
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	// Store token in database
	jwtToken := &models.JWTToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	err = s.jwtTokenRepo.Create(jwtToken)
	if err != nil {
		return "", fmt.Errorf("failed to store token in database: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns the user
// Always fetches fresh user data to ensure CanAccessData is current
func (s *AuthService) ValidateJWT(tokenString string) (*models.User, error) {
	// Parse and validate JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid user_id in token")
	}

	// Always fetch fresh user data to ensure CanAccessData is current
	// This ensures access changes take effect immediately
	user, err := s.userRepo.FindByID(uint(userID))
	if err != nil {
		return nil, err
	}

	// Check if token is revoked (validate against database)
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])
	jwtToken, err := s.jwtTokenRepo.FindByHash(tokenHash)
	if err != nil {
		// Token not found in database
		return nil, fmt.Errorf("token not found")
	}
	if jwtToken.RevokedAt != nil {
		// Token is revoked
		return nil, fmt.Errorf("token is revoked")
	}
	if jwtToken.ExpiresAt.Before(time.Now()) {
		// Token is expired
		return nil, fmt.Errorf("token is expired")
	}

	// Cache for 30 seconds (shorter cache for access control changes to take effect faster)
	if s.cacheService != nil {
		cacheKey := JWTCacheKey(tokenString)
		s.cacheService.SetJSON(cacheKey, user, 30*time.Second)
	}

	return user, nil
}

// CreateAPIKey creates a new API key for a user
func (s *AuthService) CreateAPIKey(userID uint, name string) (string, error) {
	key, hashed, err := s.GenerateAPIKey()
	if err != nil {
		return "", err
	}

	apiKey := &models.APIKey{
		UserID:  userID,
		KeyHash: hashed,
		Name:    name,
	}

	err = s.apiKeyRepo.Create(apiKey)
	if err != nil {
		return "", err
	}

	return key, nil
}

// RevokeAPIKey revokes an API key
func (s *AuthService) RevokeAPIKey(keyID uint) error {
	err := s.apiKeyRepo.Revoke(keyID)
	return err
}

// InvalidateCache invalidates cached auth data
func (s *AuthService) InvalidateCache(apiKeyHash, jwtTokenHash string) {
	if s.cacheService == nil {
		return
	}
	if apiKeyHash != "" {
		s.cacheService.DeletePattern(APIKeyCacheKey("*"))
	}
	if jwtTokenHash != "" {
		s.cacheService.DeletePattern(JWTCacheKey("*"))
	}
}

// InvalidateUserCache invalidates all cached auth data for a specific user
// This should be called when user access is updated
func (s *AuthService) InvalidateUserCache(userID uint) {
	if s.cacheService == nil {
		return
	}
	// Note: Full cache invalidation for a user would require tracking all keys
	// For now, we rely on short cache times (30s for JWT) to ensure changes take effect quickly
	// In production, you might want to implement a more sophisticated cache invalidation strategy
}
