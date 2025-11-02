package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v57/github"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/services"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"
)

type AuthHandler struct {
	authService *services.AuthService
	userService *services.UserService
	cfg         *config.Config
	oauthConfig *oauth2.Config
}

func NewAuthHandler(authService *services.AuthService, userService *services.UserService, cfg *config.Config) *AuthHandler {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		RedirectURL:  cfg.OAuthRedirectURL,
		Scopes:       []string{"user:email"},
		Endpoint:     githubOAuth.Endpoint,
	}

	return &AuthHandler{
		authService: authService,
		userService: userService,
		cfg:         cfg,
		oauthConfig: oauthConfig,
	}
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

	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user":  dbUser,
	})
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
