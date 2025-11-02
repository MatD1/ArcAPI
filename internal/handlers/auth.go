package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v57/github"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"
)

type AuthHandler struct {
	authService  *services.AuthService
	userService  *services.UserService
	cfg          *config.Config
	oauthConfig  *oauth2.Config
	apiKeyRepo   *repository.APIKeyRepository
	tempTokens   map[string]OAuthTokenData
	tempTokensMu sync.RWMutex
}

type OAuthTokenData struct {
	Token     string
	User      interface{}
	APIKey    string
	CreatedAt time.Time
}

func NewAuthHandler(
	authService *services.AuthService,
	userService *services.UserService,
	cfg *config.Config,
	apiKeyRepo *repository.APIKeyRepository,
) *AuthHandler {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		RedirectURL:  cfg.OAuthRedirectURL,
		Scopes:       []string{"user:email"},
		Endpoint:     githubOAuth.Endpoint,
	}

	handler := &AuthHandler{
		authService: authService,
		userService: userService,
		cfg:         cfg,
		oauthConfig: oauthConfig,
		apiKeyRepo:  apiKeyRepo,
		tempTokens:  make(map[string]OAuthTokenData),
	}

	// Cleanup old tokens periodically
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			handler.cleanupOldTokens()
		}
	}()

	return handler
}

func (h *AuthHandler) cleanupOldTokens() {
	h.tempTokensMu.Lock()
	defer h.tempTokensMu.Unlock()

	now := time.Now()
	for token, data := range h.tempTokens {
		if now.Sub(data.CreatedAt) > 10*time.Minute {
			delete(h.tempTokens, token)
		}
	}
}

func (h *AuthHandler) generateTempToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// ExchangeTempToken exchanges a temporary token for actual auth data
func (h *AuthHandler) ExchangeTempToken(c *gin.Context) {
	tempToken := c.Query("token")
	if tempToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing token"})
		return
	}

	h.tempTokensMu.RLock()
	data, exists := h.tempTokens[tempToken]
	h.tempTokensMu.RUnlock()

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Delete token after use (one-time use)
	h.tempTokensMu.Lock()
	delete(h.tempTokens, tempToken)
	h.tempTokensMu.Unlock()

	response := gin.H{
		"token": data.Token,
		"user":  data.User,
	}
	if data.APIKey != "" {
		response["api_key"] = data.APIKey
		response["api_key_warning"] = "Save this API key now. You won't be able to see it again."
	}

	c.JSON(http.StatusOK, response)
}

// GitHubLogin initiates GitHub OAuth flow
func (h *AuthHandler) GitHubLogin(c *gin.Context) {
	if !h.cfg.IsOAuthEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OAuth is disabled"})
		return
	}

	state := c.Query("state")
	if state == "" {
		state = "random-state" // In production, generate secure random state
	}

	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GitHubCallback handles GitHub OAuth callback
func (h *AuthHandler) GitHubCallback(c *gin.Context) {
	if !h.cfg.IsOAuthEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OAuth is disabled"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	ctx := c.Request.Context()
	token, err := h.oauthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from GitHub
	oauthClient := h.oauthConfig.Client(ctx, token)
	client := github.NewClient(oauthClient)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	// Get user email
	emails, _, err := client.Users.ListEmails(ctx, nil)
	var email string
	if err == nil && len(emails) > 0 {
		for _, e := range emails {
			if e.GetPrimary() {
				email = e.GetEmail()
				break
			}
		}
		if email == "" && len(emails) > 0 {
			email = emails[0].GetEmail()
		}
	}
	if email == "" {
		email = user.GetLogin() + "@users.noreply.github.com"
	}

	// Create or update user
	githubID := user.GetLogin()
	dbUser, err := h.userService.CreateOrUpdateFromGithub(githubID, email, user.GetLogin())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate JWT token
	jwtToken, err := h.authService.GenerateJWT(dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Auto-create API key if user doesn't have one (for OAuth users)
	// Check if user has any active API keys
	apiKeys, _ := h.apiKeyRepo.FindByUserID(dbUser.ID)
	hasActiveKey := false
	for _, key := range apiKeys {
		if key.RevokedAt == nil {
			hasActiveKey = true
			break
		}
	}

	var apiKey string
	if !hasActiveKey {
		// Create a default API key for OAuth users
		newKey, err := h.authService.CreateAPIKey(dbUser.ID, "OAuth Auto-Generated")
		if err == nil {
			apiKey = newKey
		}
	}

	// Generate temporary token for frontend to exchange
	tempToken := h.generateTempToken()

	// Store token data temporarily (in production, use Redis with TTL)
	h.tempTokensMu.Lock()
	h.tempTokens[tempToken] = OAuthTokenData{
		Token:     jwtToken,
		User:      dbUser,
		APIKey:    apiKey,
		CreatedAt: time.Now(),
	}
	h.tempTokensMu.Unlock()

	// Redirect to frontend callback with temp token
	callbackURL := h.cfg.FrontendCallbackURL
	if callbackURL == "" {
		callbackURL = "http://localhost:8080/dashboard/api/auth/github/callback/"
	}
	c.Redirect(http.StatusFound, callbackURL+"?token="+tempToken)
}

// LoginWithAPIKey authenticates with API key and returns JWT
func (h *AuthHandler) LoginWithAPIKey(c *gin.Context) {
	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate API key
	apiKey, err := h.authService.ValidateAPIKey(req.APIKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	// Get user
	user, err := h.userService.GetByID(apiKey.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	// Generate JWT token
	jwtToken, err := h.authService.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user":  user,
	})
}
