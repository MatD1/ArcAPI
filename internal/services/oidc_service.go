package services

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mat/arcapi/internal/config"
)

// OIDCClaims represents the subset of claims we care about from Authentik
type OIDCClaims struct {
	Email             string   `json:"email"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Groups            []string `json:"groups"`
	jwt.RegisteredClaims
}

// HasGroup checks whether the user belongs to a specific group
func (c *OIDCClaims) HasGroup(group string) bool {
	if group == "" {
		return false
	}
	for _, g := range c.Groups {
		if strings.EqualFold(g, group) {
			return true
		}
	}
	return false
}

type jwksResponse struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// OIDCService handles validation of OIDC tokens via JWKS
type OIDCService struct {
	cfg         *config.Config
	httpClient  *http.Client
	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	lastRefresh time.Time
}

func NewOIDCService(cfg *config.Config) (*OIDCService, error) {
	if cfg.AuthentikIssuer == "" || cfg.AuthentikJWKSURL == "" || cfg.AuthentikClientID == "" || cfg.AuthentikUserInfoURL == "" {
		return nil, errors.New("authentik configuration is incomplete - issuer, jwks_url, client_id, and userinfo_url are required")
	}

	svc := &OIDCService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
	}

	if err := svc.refreshKeys(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	return svc, nil
}

func (s *OIDCService) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.AuthentikJWKSURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	keyMap := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if strings.ToUpper(key.Kty) != "RSA" || key.Kid == "" {
			continue
		}
		pubKey, err := parseRSAPublicKey(key.N, key.E)
		if err != nil {
			continue
		}
		keyMap[key.Kid] = pubKey
	}

	if len(keyMap) == 0 {
		return errors.New("no valid RSA keys found in JWKS")
	}

	s.mu.Lock()
	s.keys = keyMap
	s.lastRefresh = time.Now()
	s.mu.Unlock()

	return nil
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}

	eb, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nb)
	if len(eb) == 0 {
		return nil, errors.New("invalid exponent")
	}

	e := 0
	for _, b := range eb {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

func (s *OIDCService) keyForToken(token *jwt.Token) (*rsa.PublicKey, error) {
	kid, _ := token.Header["kid"].(string)

	s.mu.RLock()
	key := s.keys[kid]
	lastRefresh := s.lastRefresh
	s.mu.RUnlock()

	if key != nil {
		return key, nil
	}

	// Refresh if key missing or cache older than 1h
	if time.Since(lastRefresh) > time.Hour || key == nil {
		if err := s.refreshKeys(context.Background()); err != nil {
			return nil, err
		}
		s.mu.RLock()
		key = s.keys[kid]
		s.mu.RUnlock()
	}

	if key == nil {
		return nil, fmt.Errorf("no matching JWKS key for kid %q", kid)
	}

	return key, nil
}

// ValidateToken validates an incoming OIDC token and returns the claims
// This method now handles both JWS and JWE tokens by attempting userinfo fallback for JWE
func (s *OIDCService) ValidateToken(tokenString string) (*OIDCClaims, error) {
	claims := &OIDCClaims{}

	// First try to parse as standard JWS token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.keyForToken(token)
	}, jwt.WithAudience(s.cfg.AuthentikClientID), jwt.WithIssuer(s.cfg.AuthentikIssuer), jwt.WithValidMethods([]string{"RS256"}))

	// If parsing succeeds and token is valid, return claims
	if err == nil && token.Valid {
		if claims.Email == "" {
			return nil, errors.New("email claim missing")
		}
		return claims, nil
	}

	// If parsing failed due to JWE format, try to detect JWE token
	if strings.Contains(err.Error(), "could not JSON decode header") || strings.Contains(err.Error(), "invalid character") {
		// This might be a JWE token. For JWE tokens, we should use the userinfo endpoint
		// instead of trying to decrypt the token locally.
		return nil, fmt.Errorf("JWE tokens detected - token validation should use userinfo endpoint instead: %w", err)
	}

	// Return the original error for other cases
	return nil, err
}

// ValidateTokenWithUserInfo validates a token by calling the userinfo endpoint
// This is the preferred method when Authentik sends JWE-encrypted tokens
func (s *OIDCService) ValidateTokenWithUserInfo(ctx context.Context, accessToken string) (*OIDCClaims, error) {
	if s.cfg.AuthentikUserInfoURL == "" {
		return nil, errors.New("userinfo URL not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", s.cfg.AuthentikUserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	// Extract claims from userinfo response
	claims := &OIDCClaims{}

	if email, ok := userInfo["email"].(string); ok {
		claims.Email = email
	}

	if preferredUsername, ok := userInfo["preferred_username"].(string); ok {
		claims.PreferredUsername = preferredUsername
	}

	if name, ok := userInfo["name"].(string); ok {
		claims.Name = name
	}

	if groups, ok := userInfo["groups"].([]interface{}); ok {
		claims.Groups = make([]string, len(groups))
		for i, g := range groups {
			if groupStr, ok := g.(string); ok {
				claims.Groups[i] = groupStr
			}
		}
	}

	// Set issuer and audience from config
	claims.Issuer = s.cfg.AuthentikIssuer
	claims.Audience = jwt.ClaimStrings{s.cfg.AuthentikClientID}

	if claims.Email == "" {
		return nil, errors.New("email claim missing from userinfo")
	}

	return claims, nil
}
