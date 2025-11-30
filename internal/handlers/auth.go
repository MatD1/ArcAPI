package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v57/github"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/mat/arcapi/internal/services"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"
)

type AuthHandler struct {
	authService        *services.AuthService
	userService        *services.UserService
	cfg                *config.Config
	githubOAuthConfig  *oauth2.Config
	discordOAuthConfig *oauth2.Config
	oidcService        *services.OIDCService
	apiKeyRepo         *repository.APIKeyRepository
	tempTokens         map[string]OAuthTokenData
	tempTokensMu       sync.RWMutex
}

type OAuthTokenData struct {
	Token     string
	User      interface{}
	APIKey    string
	CreatedAt time.Time
}

// OAuthState represents the state passed through OAuth flow
type OAuthState struct {
	Redirect            string    `json:"redirect"`       // Deep link URL for mobile, or web callback
	Client              string    `json:"client"`         // "mobile" or "web"
	CSRFToken           string    `json:"csrf_token"`     // CSRF protection token
	Timestamp           time.Time `json:"timestamp"`      // State creation time
	Mode                string    `json:"mode,omitempty"` // "pkce" for authorization code flow, else empty for temp token flow
	CodeChallenge       string    `json:"code_challenge,omitempty"`
	CodeChallengeMethod string    `json:"code_challenge_method,omitempty"`
	ExternalState       string    `json:"external_state,omitempty"` // Original state from client (passed back on redirect)
}

const (
	ClientMobile = "mobile"
	ClientWeb    = "web"
	StateExpiry  = 10 * time.Minute
)

func NewAuthHandler(
	authService *services.AuthService,
	userService *services.UserService,
	cfg *config.Config,
	apiKeyRepo *repository.APIKeyRepository,
	oidcService *services.OIDCService,
) *AuthHandler {
	var githubOAuthConfig *oauth2.Config
	if cfg.IsGitHubOAuthEnabled() {
		githubOAuthConfig = &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.OAuthRedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     githubOAuth.Endpoint,
		}
	}

	var discordOAuthConfig *oauth2.Config
	if cfg.IsDiscordOAuthEnabled() {
		discordOAuthConfig = &oauth2.Config{
			ClientID:     cfg.DiscordClientID,
			ClientSecret: cfg.DiscordClientSecret,
			RedirectURL:  cfg.DiscordRedirectURL,
			Scopes:       []string{"identify", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
		}
	}

	handler := &AuthHandler{
		authService:        authService,
		userService:        userService,
		cfg:                cfg,
		githubOAuthConfig:  githubOAuthConfig,
		discordOAuthConfig: discordOAuthConfig,
		oidcService:        oidcService,
		apiKeyRepo:         apiKeyRepo,
		tempTokens:         make(map[string]OAuthTokenData),
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

// generateCSRFToken generates a random CSRF token
func (h *AuthHandler) generateCSRFToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// encodeState encodes OAuthState to base64 URL-encoded string
func encodeState(state *OAuthState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// decodeState decodes base64 URL-encoded string to OAuthState
func decodeState(encoded string) (*OAuthState, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	var state OAuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// validateRedirectURL validates that the redirect URL is safe
// For mobile: must start with "arcdb://"
// For web: must be http(s) URL
func validateRedirectURL(redirectURL, client string) error {
	if redirectURL == "" {
		return fmt.Errorf("redirect URL is required")
	}

	parsed, err := url.Parse(redirectURL)
	if err != nil {
		return fmt.Errorf("invalid redirect URL: %w", err)
	}

	if client == ClientMobile {
		// Mobile deep link must start with arcdb://
		if parsed.Scheme != "arcdb" {
			return fmt.Errorf("mobile redirect URL must use arcdb:// scheme, got: %s", parsed.Scheme)
		}
	} else {
		// Web redirect must be http or https
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("web redirect URL must use http:// or https:// scheme, got: %s", parsed.Scheme)
		}
	}

	return nil
}

// validateState validates state expiration and CSRF token
func validateState(state *OAuthState) error {
	// Check expiration
	if time.Since(state.Timestamp) > StateExpiry {
		return fmt.Errorf("state expired")
	}

	// CSRF token should not be empty
	if state.CSRFToken == "" {
		return fmt.Errorf("missing CSRF token")
	}

	return nil
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
		response["api_key_warning"] = "A new API key has been generated for this login. Save it now if you want to use it. You won't be able to see it again. If you already have API keys, you can continue using those instead."
	}

	c.JSON(http.StatusOK, response)
}

// GitHubLogin initiates GitHub OAuth flow
func (h *AuthHandler) GitHubLogin(c *gin.Context) {
	if !h.cfg.IsGitHubOAuthEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GitHub OAuth is not configured"})
		return
	}

	// Build OAuth callback URL from request if not set in config
	oauthCallbackURL := h.cfg.OAuthRedirectURL
	if oauthCallbackURL == "" || strings.Contains(oauthCallbackURL, "localhost") {
		scheme := "https"
		// Check X-Forwarded-Proto header first (for proxies like Railway)
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS == nil {
			scheme = "http"
		}
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}
		oauthCallbackURL = fmt.Sprintf("%s://%s/api/v1/auth/github/callback", scheme, host)
	}

	// Update OAuth config with dynamic redirect URL
	oauthConfig := *h.githubOAuthConfig
	oauthConfig.RedirectURL = oauthCallbackURL

	// Extract redirect and client parameters from query
	redirectParam := strings.TrimSpace(c.Query("redirect"))
	clientParam := strings.TrimSpace(c.Query("client"))

	// Default to web if client not specified
	if clientParam == "" {
		clientParam = ClientWeb
	}

	var state *OAuthState
	var stateString string

	// If mobile client with redirect, create state with mobile deep link
	if clientParam == ClientMobile {
		if redirectParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mobile client requires redirect parameter"})
			return
		}
		// Validate mobile redirect URL
		if err := validateRedirectURL(redirectParam, ClientMobile); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid redirect URL: %v", err)})
			return
		}

		state = &OAuthState{
			Redirect:  redirectParam,
			Client:    ClientMobile,
			CSRFToken: h.generateCSRFToken(),
			Timestamp: time.Now(),
		}
	} else if clientParam == ClientWeb {
		// For web, use default frontend callback URL
		webCallbackURL := h.cfg.FrontendCallbackURL
		if webCallbackURL == "" || strings.Contains(webCallbackURL, "localhost") {
			scheme := "https"
			if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			} else if c.Request.TLS == nil {
				scheme = "http"
			}
			host := c.GetHeader("X-Forwarded-Host")
			if host == "" {
				host = c.Request.Host
			}
			webCallbackURL = fmt.Sprintf("%s://%s/dashboard/api/auth/github/callback/", scheme, host)
		}

		state = &OAuthState{
			Redirect:  webCallbackURL,
			Client:    ClientWeb,
			CSRFToken: h.generateCSRFToken(),
			Timestamp: time.Now(),
		}
	} else {
		// Invalid client type
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid client type: %s. Must be 'mobile' or 'web'", clientParam)})
		return
	}

	// Encode state
	var err error
	stateString, err = encodeState(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode state"})
		return
	}

	url := oauthConfig.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
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
	token, err := h.githubOAuthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from GitHub
	oauthClient := h.githubOAuthConfig.Client(ctx, token)
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

	// Decode state to check if user was created via mobile app
	var createdViaApp bool
	stateParam := c.Query("state")
	if stateParam != "" {
		if state, err := decodeState(stateParam); err == nil {
			createdViaApp = (state.Client == ClientMobile)
		}
	}

	// Create or update user
	githubID := user.GetLogin()
	dbUser, err := h.userService.CreateOrUpdateFromGithub(githubID, email, user.GetLogin(), createdViaApp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	// Validate user was created/updated successfully
	if dbUser == nil || dbUser.ID == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user after creation"})
		return
	}

	// Generate JWT token
	jwtToken, err := h.authService.GenerateJWT(dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token", "details": err.Error()})
		return
	}

	// Don't auto-create API keys for OAuth users - they can use JWT for read operations
	// API keys are only needed for write operations (or can be created manually by admins)
	var apiKey string
	// Check if user has any active API keys - only create if they don't have any
	apiKeys, _ := h.apiKeyRepo.FindByUserID(dbUser.ID)
	hasActiveKey := false
	for _, key := range apiKeys {
		if key.RevokedAt == nil {
			hasActiveKey = true
			break
		}
	}

	// Only create an API key if user doesn't have any (first login convenience)
	if !hasActiveKey {
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

	// Decode and validate state from OAuth callback (we already decoded it earlier, but need to validate)
	if stateParam == "" {
		// Cleanup temp token on error
		h.tempTokensMu.Lock()
		delete(h.tempTokens, tempToken)
		h.tempTokensMu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing state parameter"})
		return
	}

	state, err := decodeState(stateParam)
	if err != nil {
		// Cleanup temp token on error
		h.tempTokensMu.Lock()
		delete(h.tempTokens, tempToken)
		h.tempTokensMu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid state: %v", err)})
		return
	}

	// Validate state expiration and CSRF token
	if err := validateState(state); err != nil {
		// Cleanup temp token on error
		h.tempTokensMu.Lock()
		delete(h.tempTokens, tempToken)
		h.tempTokensMu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("State validation failed: %v", err)})
		return
	}

	// Determine callback destination based on client type
	var callbackURL string

	if state.Client == ClientMobile {
		// Mobile: redirect to web page first, which will then redirect to deep link
		deepLinkURL := state.Redirect
		// Validate redirect URL one more time for safety
		if err := validateRedirectURL(deepLinkURL, ClientMobile); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid redirect URL: %v", err)})
			return
		}

		// Build web callback URL that will redirect to the deep link
		scheme := "https"
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS == nil {
			scheme = "http"
		}
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}
		callbackURL = fmt.Sprintf("%s://%s/auth/mobile-callback?token=%s&redirect=%s",
			scheme,
			host,
			url.QueryEscape(tempToken),
			url.QueryEscape(deepLinkURL),
		)
		c.Redirect(http.StatusFound, callbackURL)
		return
	} else {
		// Web: use state redirect (which should be the frontend callback URL)
		callbackURL = state.Redirect
		// Fallback to default if somehow empty
		if callbackURL == "" {
			callbackURL = h.cfg.FrontendCallbackURL
			if callbackURL == "" || strings.Contains(callbackURL, "localhost") {
				scheme := "https"
				if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
					scheme = proto
				} else if c.Request.TLS == nil {
					scheme = "http"
				}
				host := c.GetHeader("X-Forwarded-Host")
				if host == "" {
					host = c.Request.Host
				}
				callbackURL = fmt.Sprintf("%s://%s/dashboard/api/auth/github/callback/", scheme, host)
			}
		}
	}

	// Append token appropriately
	sep := "?"
	if strings.Contains(callbackURL, "?") {
		sep = "&"
	}
	finalRedirectURL := callbackURL + sep + "token=" + tempToken
	c.Redirect(http.StatusFound, finalRedirectURL)
}

// MobileCallbackPage handles the web page that redirects mobile clients to deep links
func (h *AuthHandler) MobileCallbackPage(c *gin.Context) {
	token := c.Query("token")
	redirect := c.Query("redirect")

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing token parameter"})
		return
	}

	if redirect == "" {
		redirect = "arcdb://auth/callback"
	}

	// Validate redirect URL scheme
	parsed, err := url.Parse(redirect)
	if err != nil || parsed.Scheme != "arcdb" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid redirect URL"})
		return
	}

	// Build deep link with token
	deepLink := fmt.Sprintf("%s?token=%s", redirect, url.QueryEscape(token))

	// Render HTML page that redirects to deep link
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Redirecting...</title>
    <script>
        (function() {
            const deepLink = %q;
            // Try to redirect immediately
            window.location.href = deepLink;
            // Fallback: if redirect doesn't work after 1 second, show message
            setTimeout(function() {
                document.body.innerHTML = '<div style="text-align: center; padding: 2rem;"><h2>Opening app...</h2><p>If the app doesn\'t open, <a href="' + deepLink + '">click here</a></p></div>';
            }, 1000);
        })();
    </script>
</head>
<body style="font-family: system-ui, -apple-system, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #1a1a1a; color: #fff;">
    <p>Redirecting to app...</p>
</body>
</html>`, deepLink)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// DiscordLogin initiates Discord OAuth flow
func (h *AuthHandler) DiscordLogin(c *gin.Context) {
	if !h.cfg.IsDiscordOAuthEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Discord OAuth is not configured"})
		return
	}

	// Build OAuth callback URL from request if not set in config
	oauthCallbackURL := h.cfg.DiscordRedirectURL
	if oauthCallbackURL == "" || strings.Contains(oauthCallbackURL, "localhost") {
		scheme := "https"
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS == nil {
			scheme = "http"
		}
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}
		oauthCallbackURL = fmt.Sprintf("%s://%s/api/v1/auth/discord/callback", scheme, host)
	}

	// Update OAuth config with dynamic redirect URL
	oauthConfig := *h.discordOAuthConfig
	oauthConfig.RedirectURL = oauthCallbackURL

	// Extract redirect and client parameters from query
	redirectParam := strings.TrimSpace(c.Query("redirect"))
	clientParam := strings.TrimSpace(c.Query("client"))

	// Default to web if client not specified
	if clientParam == "" {
		clientParam = ClientWeb
	}

	var state *OAuthState
	var stateString string

	// If mobile client with redirect, create state with mobile deep link
	if clientParam == ClientMobile {
		if redirectParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mobile client requires redirect parameter"})
			return
		}
		if err := validateRedirectURL(redirectParam, ClientMobile); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid redirect URL: %v", err)})
			return
		}

		state = &OAuthState{
			Redirect:  redirectParam,
			Client:    ClientMobile,
			CSRFToken: h.generateCSRFToken(),
			Timestamp: time.Now(),
		}
	} else if clientParam == ClientWeb {
		webCallbackURL := h.cfg.FrontendCallbackURL
		if webCallbackURL == "" || strings.Contains(webCallbackURL, "localhost") {
			scheme := "https"
			if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			} else if c.Request.TLS == nil {
				scheme = "http"
			}
			host := c.GetHeader("X-Forwarded-Host")
			if host == "" {
				host = c.Request.Host
			}
			webCallbackURL = fmt.Sprintf("%s://%s/dashboard/api/auth/discord/callback/", scheme, host)
		}

		state = &OAuthState{
			Redirect:  webCallbackURL,
			Client:    ClientWeb,
			CSRFToken: h.generateCSRFToken(),
			Timestamp: time.Now(),
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid client type: %s. Must be 'mobile' or 'web'", clientParam)})
		return
	}

	// Encode state
	var err error
	stateString, err = encodeState(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode state"})
		return
	}

	url := oauthConfig.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// DiscordCallback handles Discord OAuth callback
func (h *AuthHandler) DiscordCallback(c *gin.Context) {
	if !h.cfg.IsDiscordOAuthEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Discord OAuth is not configured"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	ctx := c.Request.Context()
	token, err := h.discordOAuthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from Discord API
	oauthClient := h.discordOAuthConfig.Client(ctx, token)
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	resp, err := oauthClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()

	var discordUser struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		Email         string `json:"email"`
		Discriminator string `json:"discriminator"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&discordUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode user info"})
		return
	}

	// Use email if available, otherwise use username
	email := discordUser.Email
	if email == "" {
		email = discordUser.Username + "@discord.noreply.com"
	}

	// Decode state to check if user was created via mobile app
	var createdViaApp bool
	stateParam := c.Query("state")
	if stateParam != "" {
		if state, err := decodeState(stateParam); err == nil {
			createdViaApp = (state.Client == ClientMobile)
		}
	}

	// Create or update user
	discordID := discordUser.ID
	dbUser, err := h.userService.CreateOrUpdateFromDiscord(discordID, email, discordUser.Username, createdViaApp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	// Validate user was created/updated successfully
	if dbUser == nil || dbUser.ID == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user after creation"})
		return
	}

	// Generate JWT token
	jwtToken, err := h.authService.GenerateJWT(dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token", "details": err.Error()})
		return
	}

	// Check if user has any active API keys - only create if they don't have any
	var apiKey string
	apiKeys, _ := h.apiKeyRepo.FindByUserID(dbUser.ID)
	hasActiveKey := false
	for _, key := range apiKeys {
		if key.RevokedAt == nil {
			hasActiveKey = true
			break
		}
	}

	if !hasActiveKey {
		newKey, err := h.authService.CreateAPIKey(dbUser.ID, "OAuth Auto-Generated")
		if err == nil {
			apiKey = newKey
		}
	}

	// Generate temporary token for frontend to exchange
	tempToken := h.generateTempToken()

	// Store token data temporarily
	h.tempTokensMu.Lock()
	h.tempTokens[tempToken] = OAuthTokenData{
		Token:     jwtToken,
		User:      dbUser,
		APIKey:    apiKey,
		CreatedAt: time.Now(),
	}
	h.tempTokensMu.Unlock()

	// Decode and validate state
	if stateParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing state parameter"})
		return
	}

	state, err := decodeState(stateParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid state: %v", err)})
		return
	}

	if err := validateState(state); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("State validation failed: %v", err)})
		return
	}

	// Determine callback destination based on client type
	var callbackURL string

	if state.Client == ClientMobile {
		deepLinkURL := state.Redirect
		if err := validateRedirectURL(deepLinkURL, ClientMobile); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid redirect URL: %v", err)})
			return
		}

		scheme := "https"
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS == nil {
			scheme = "http"
		}
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}
		callbackURL = fmt.Sprintf("%s://%s/auth/mobile-callback?token=%s&redirect=%s",
			scheme,
			host,
			url.QueryEscape(tempToken),
			url.QueryEscape(deepLinkURL),
		)
		c.Redirect(http.StatusFound, callbackURL)
		return
	} else {
		callbackURL = state.Redirect
		if callbackURL == "" {
			callbackURL = h.cfg.FrontendCallbackURL
			if callbackURL == "" || strings.Contains(callbackURL, "localhost") {
				scheme := "https"
				if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
					scheme = proto
				} else if c.Request.TLS == nil {
					scheme = "http"
				}
				host := c.GetHeader("X-Forwarded-Host")
				if host == "" {
					host = c.Request.Host
				}
				callbackURL = fmt.Sprintf("%s://%s/dashboard/api/auth/discord/callback/", scheme, host)
			}
		}
	}

	// Append token appropriately
	sep := "?"
	if strings.Contains(callbackURL, "?") {
		sep = "&"
	}
	finalRedirectURL := callbackURL + sep + "token=" + tempToken
	c.Redirect(http.StatusFound, finalRedirectURL)
}

// TokenExchange exchanges authorization code + code_verifier for JWT + refresh token (PKCE flow)
func (h *AuthHandler) TokenExchange(c *gin.Context) {
	var req struct {
		Code         string `json:"code" binding:"required"`
		CodeVerifier string `json:"code_verifier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	jwt, refresh, user, err := h.authService.ExchangeAuthorizationCode(req.Code, req.CodeVerifier)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": jwt, "refresh_token": refresh, "user": user})
}

// AuthentikTokenExchange handles PKCE code exchanges for Authentik with robust error handling
func (h *AuthHandler) AuthentikTokenExchange(c *gin.Context) {
	// Create a context with timeout for the entire operation (45 seconds total)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	// Update request context
	c.Request = c.Request.WithContext(ctx)

	// Phase 1: Validate configuration and inputs
	if err := h.validateAuthentikConfig(); err != nil {
		h.logAndRespond(c, http.StatusServiceUnavailable, "configuration_error", err.Error(), nil)
		return
	}

	var req struct {
		Code         string `json:"code" binding:"required"`
		CodeVerifier string `json:"code_verifier" binding:"required"`
		RedirectURI  string `json:"redirect_uri" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logAndRespond(c, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", gin.H{"details": err.Error()})
		return
	}

	// Validate input lengths and format
	if err := h.validateTokenExchangeInputs(req); err != nil {
		h.logAndRespond(c, http.StatusBadRequest, "invalid_input", err.Error(), nil)
		return
	}

	// Phase 2: Exchange authorization code for tokens from Authentik
	tokenResponse, err := h.exchangeAuthentikCode(ctx, req)
	if err != nil {
		h.logAndRespond(c, http.StatusBadGateway, "token_exchange_failed", err.Error(), nil)
		return
	}

	// Phase 3: Validate the received token
	claims, err := h.validateAuthentikToken(tokenResponse)
	if err != nil {
		h.logAndRespond(c, http.StatusUnauthorized, "token_validation_failed", err.Error(), nil)
		return
	}

	// Phase 4: Sync/create user in database
	user, err := h.syncUserWithRetry(ctx, claims)
	if err != nil {
		h.logAndRespond(c, http.StatusInternalServerError, "user_sync_failed", err.Error(), nil)
		return
	}

	// Phase 5: Issue application tokens
	appTokens, err := h.issueApplicationTokens(ctx, user)
	if err != nil {
		h.logAndRespond(c, http.StatusInternalServerError, "token_issuance_failed", err.Error(), nil)
		return
	}

	// Phase 6: Return successful response
	expiresIn := h.cfg.JWTExpiryHours * 3600
	c.JSON(http.StatusOK, gin.H{
		"token":         appTokens.JWT,
		"id_token":      appTokens.JWT,
		"refresh_token": appTokens.RefreshToken,
		"expires_in":    expiresIn,
		"user":          user,
	})
}

// validateAuthentikConfig validates that Authentik is properly configured
func (h *AuthHandler) validateAuthentikConfig() error {
	if !h.cfg.AuthentikEnabled {
		return fmt.Errorf("authentik authentication is not enabled")
	}
	if h.oidcService == nil {
		return fmt.Errorf("OIDC service not initialized")
	}
	if h.cfg.AuthentikTokenURL == "" {
		return fmt.Errorf("authentik token URL not configured")
	}
	if h.cfg.AuthentikClientID == "" {
		return fmt.Errorf("authentik client ID not configured")
	}
	return nil
}

// validateTokenExchangeInputs validates the input parameters
func (h *AuthHandler) validateTokenExchangeInputs(req struct {
	Code         string `json:"code" binding:"required"`
	CodeVerifier string `json:"code_verifier" binding:"required"`
	RedirectURI  string `json:"redirect_uri" binding:"required"`
}) error {
	if len(req.Code) < 10 {
		return fmt.Errorf("authorization code too short")
	}
	if len(req.CodeVerifier) < 43 {
		return fmt.Errorf("code verifier too short (minimum 43 characters)")
	}
	if len(req.RedirectURI) < 10 {
		return fmt.Errorf("redirect URI too short")
	}
	if !strings.HasPrefix(req.RedirectURI, "https://") {
		return fmt.Errorf("redirect URI must use HTTPS")
	}
	return nil
}

// exchangeAuthentikCode exchanges the authorization code for tokens
func (h *AuthHandler) exchangeAuthentikCode(ctx context.Context, req struct {
	Code         string `json:"code" binding:"required"`
	CodeVerifier string `json:"code_verifier" binding:"required"`
	RedirectURI  string `json:"redirect_uri" binding:"required"`
}) (*authentikTokenResponse, error) {
	// Create form data
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", req.Code)
	form.Set("redirect_uri", req.RedirectURI)
	form.Set("client_id", h.cfg.AuthentikClientID)
	form.Set("code_verifier", req.CodeVerifier)
	if secret := strings.TrimSpace(h.cfg.AuthentikClientSecret); secret != "" {
		form.Set("client_secret", secret)
	}

	// Create HTTP request with timeout
	reqBody := strings.NewReader(form.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.AuthentikTokenURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", "ArcAPI/1.0")

	// Execute request with 15 second timeout
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call authentik: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentik returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Parse response
	var tokenResp authentikTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// validateAuthentikToken validates the received token
func (h *AuthHandler) validateAuthentikToken(tokenResp *authentikTokenResponse) (*services.OIDCClaims, error) {
	// Prefer id_token, fall back to access_token
	tokenToValidate := tokenResp.IDToken
	if tokenToValidate == "" {
		tokenToValidate = tokenResp.AccessToken
	}
	if tokenToValidate == "" {
		return nil, fmt.Errorf("no valid token received from authentik")
	}

	// Basic JWT format validation
	if !strings.Contains(tokenToValidate, ".") {
		return nil, fmt.Errorf("received token is not a valid JWT format")
	}

	segments := strings.Split(tokenToValidate, ".")
	if len(segments) != 3 {
		return nil, fmt.Errorf("received token has invalid JWT structure (%d segments, expected 3)", len(segments))
	}

	// Validate token with OIDC service
	claims, err := h.oidcService.ValidateToken(tokenToValidate)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Ensure required claims are present
	if claims.Email == "" {
		return nil, fmt.Errorf("token missing required email claim")
	}

	return claims, nil
}

// syncUserWithRetry syncs the OIDC user with retry logic for database issues
func (h *AuthHandler) syncUserWithRetry(ctx context.Context, claims *services.OIDCClaims) (*models.User, error) {
	// Try user sync with retry on database errors
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		user, err := h.authService.SyncOIDCUser(claims)
		if err == nil {
			return user, nil
		}

		lastErr = err

		// Check if this is a retryable database error
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "connection refused") &&
		   !strings.Contains(errMsg, "connection") &&
		   !strings.Contains(errMsg, "timeout") &&
		   !strings.Contains(errMsg, "no connection") &&
		   !strings.Contains(errMsg, "bad connection") {
			// Not a database error, don't retry
			break
		}

		// Wait before retry (exponential backoff)
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
				// Continue to retry
			}
		}
	}

	return nil, fmt.Errorf("user synchronization failed after %d attempts: %w", maxRetries, lastErr)
}

// issueApplicationTokens issues JWT and refresh tokens for the user
func (h *AuthHandler) issueApplicationTokens(ctx context.Context, user *models.User) (*applicationTokens, error) {
	// Try token issuance with retry on database errors
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		jwtToken, refreshToken, err := h.authService.IssueTokensForUser(user)
		if err == nil {
			return &applicationTokens{
				JWT:          jwtToken,
				RefreshToken: refreshToken,
			}, nil
		}

		lastErr = err

		// Check if this is a retryable database error
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "connection refused") &&
		   !strings.Contains(errMsg, "connection") &&
		   !strings.Contains(errMsg, "timeout") &&
		   !strings.Contains(errMsg, "no connection") &&
		   !strings.Contains(errMsg, "bad connection") {
			// Not a database error, don't retry
			break
		}

		// Wait before retry (exponential backoff)
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
				// Continue to retry
			}
		}
	}

	return nil, fmt.Errorf("token issuance failed after %d attempts: %w", maxRetries, lastErr)
}

// logAndRespond logs the error and sends a consistent JSON response
func (h *AuthHandler) logAndRespond(c *gin.Context, statusCode int, errorCode, message string, extra gin.H) {
	response := gin.H{
		"error": errorCode,
		"message": message,
	}

	// Add extra fields if provided
	for k, v := range extra {
		response[k] = v
	}

	// Add retry_after for certain error types
	if statusCode == http.StatusServiceUnavailable {
		response["retry_after"] = 2
	}

	c.JSON(statusCode, response)
}

// Structs for type safety
type authentikTokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type applicationTokens struct {
	JWT          string
	RefreshToken string
}

// RefreshToken endpoint rotates refresh token and returns new JWT + refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	jwt, newRefresh, user, err := h.authService.RefreshJWT(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": jwt, "refresh_token": newRefresh, "user": user})
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

// GetCurrentUser returns the authenticated user context
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	val, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, ok := val.(*models.User)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
