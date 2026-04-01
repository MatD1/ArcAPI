package services

import (
	crand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo         *repository.UserRepository
	apiKeyRepo       *repository.APIKeyRepository
	jwtTokenRepo     *repository.JWTTokenRepository
	authCodeRepo     *repository.AuthorizationCodeRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	cacheService     *CacheService
	cfg              *config.Config
}

func NewAuthService(
	userRepo *repository.UserRepository,
	apiKeyRepo *repository.APIKeyRepository,
	jwtTokenRepo *repository.JWTTokenRepository,
	authCodeRepo *repository.AuthorizationCodeRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	cacheService *CacheService,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		apiKeyRepo:       apiKeyRepo,
		jwtTokenRepo:     jwtTokenRepo,
		authCodeRepo:     authCodeRepo,
		refreshTokenRepo: refreshTokenRepo,
		cacheService:     cacheService,
		cfg:              cfg,
	}
}

func (s *AuthService) UserRepo() *repository.UserRepository {
	return s.userRepo
}

// IssueTokensForUser is removed - Use Supabase for tokens

// GenerateAPIKey generates a new API key and returns both the plain key and hashed version
func (s *AuthService) GenerateAPIKey() (string, string, error) {
	keyBytes := make([]byte, 32)
	_, err := crand.Read(keyBytes)
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

// JWT validation is now handled via SupabaseAuthService

// SyncSupabaseUser ensures there is a local user matching the Supabase identity
func (s *AuthService) SyncSupabaseUser(claims *SupabaseClaims) (*models.User, error) {
	if claims == nil {
		return nil, fmt.Errorf("supabase claims missing")
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return nil, fmt.Errorf("email claim missing")
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user, err = s.createSupabaseUser(claims)
			if err != nil {
				return nil, err
			}
			log.Printf("Successfully created new user from Supabase sync: %s (%s)", user.Email, user.Username)
		} else {
			return nil, err
		}
	}

	// Sync role and access status from Supabase metadata
	wasUpdated := false
	if !user.CanAccessData {
		user.CanAccessData = true
		wasUpdated = true
	}

	// Check if user is an admin in Supabase
	isAdmin := false
	if role, ok := claims.AppMetadata["role"].(string); ok && role == "admin" {
		isAdmin = true
	} else if role, ok := claims.UserMetadata["role"].(string); ok && role == "admin" {
		isAdmin = true
	} else if userRole, ok := claims.AppMetadata["user_role"].(string); ok && userRole == "admin" {
		isAdmin = true
	}

	// Log metadata for debugging if user is not detected as admin
	if !isAdmin {
		log.Printf("DEBUG: User %s metadata - AppMetadata: %v, UserMetadata: %v", user.Email, claims.AppMetadata, claims.UserMetadata)
	}

	// Promote to admin if specified in Supabase and not already admin
	if isAdmin && user.Role != models.RoleAdmin {
		user.Role = models.RoleAdmin
		wasUpdated = true
		log.Printf("Promoting user %s to admin based on Supabase metadata", user.Email)
	}

	if wasUpdated {
		if err := s.userRepo.Update(user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *AuthService) createSupabaseUser(claims *SupabaseClaims) (*models.User, error) {
	if idx := strings.Index(claims.Email, "@"); idx > 0 {
	}
	baseUsername := ""
	if idx := strings.Index(claims.Email, "@"); idx > 0 {
		baseUsername = claims.Email[:idx]
	} else {
		baseUsername = "user"
	}

	baseUsername = sanitizeUsername(baseUsername)
	if baseUsername == "" {
		baseUsername = "user"
	}

	for attempt := 0; attempt < 8; attempt++ {
		username := baseUsername
		if attempt > 0 {
			username = fmt.Sprintf("%s-%d", baseUsername, rand.Intn(100000))
		}

		user := &models.User{
			Email:         strings.ToLower(claims.Email),
			Username:      username,
			Role:          models.RoleUser, // Default to user, manual update to admin needed
			CanAccessData: true,
			CreatedViaApp: true,
		}

		if err := s.userRepo.Create(user); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				continue
			}
			return nil, err
		}

		return user, nil
	}

	return nil, fmt.Errorf("unable to create unique username for %s", claims.Email)
}

func sanitizeUsername(input string) string {
	trimmed := strings.ToLower(strings.TrimSpace(input))
	builder := strings.Builder{}
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
		case r == ' ':
			builder.WriteRune('-')
		}
	}
	result := strings.Trim(builder.String(), "-_.")
	if result == "" {
		return "user"
	}
	return result
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
}

// UpdateUserRole updates a user's role in the database
func (s *AuthService) UpdateUserRole(user *models.User, newRole models.UserRole) error {
	user.Role = newRole
	return s.userRepo.Update(user)
}
