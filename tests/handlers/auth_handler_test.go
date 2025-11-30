package handlers_test

import (
	"testing"

	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/handlers"
	"github.com/mat/arcapi/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestValidateAuthentikConfig_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		AuthentikEnabled:     true,
		AuthentikTokenURL:    "https://auth.example.com/token",
		AuthentikClientID:    "test-client-id",
	}
	oidcService := &services.OIDCService{} // Mock/mock would be needed for full testing
	handler := &handlers.AuthHandler{
		cfg:         cfg,
		oidcService: oidcService,
	}

	// Test
	err := handler.validateAuthentikConfig()

	// Assert
	assert.NoError(t, err)
}

func TestValidateAuthentikConfig_Disabled(t *testing.T) {
	// Setup
	cfg := &config.Config{AuthentikEnabled: false}
	handler := &handlers.AuthHandler{cfg: cfg}

	// Test
	err := handler.validateAuthentikConfig()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestValidateAuthentikConfig_NoOIDCService(t *testing.T) {
	// Setup
	cfg := &config.Config{
		AuthentikEnabled:     true,
		AuthentikTokenURL:    "https://auth.example.com/token",
		AuthentikClientID:    "test-client-id",
	}
	handler := &handlers.AuthHandler{cfg: cfg}

	// Test
	err := handler.validateAuthentikConfig()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC service not initialized")
}

func TestValidateTokenExchangeInputs_Valid(t *testing.T) {
	// Setup
	handler := &handlers.AuthHandler{}
	req := struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RedirectURI  string `json:"redirect_uri"`
	}{
		Code:         "valid-authorization-code",
		CodeVerifier: "valid-code-verifier-with-sufficient-length-for-pkce",
		RedirectURI:  "https://example.com/callback",
	}

	// Test
	err := handler.validateTokenExchangeInputs(req)

	// Assert
	assert.NoError(t, err)
}

func TestValidateTokenExchangeInputs_CodeTooShort(t *testing.T) {
	// Setup
	handler := &handlers.AuthHandler{}
	req := struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RedirectURI  string `json:"redirect_uri"`
	}{
		Code:         "short",
		CodeVerifier: "valid-code-verifier-with-sufficient-length-for-pkce",
		RedirectURI:  "https://example.com/callback",
	}

	// Test
	err := handler.validateTokenExchangeInputs(req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code too short")
}

func TestValidateTokenExchangeInputs_CodeVerifierTooShort(t *testing.T) {
	// Setup
	handler := &handlers.AuthHandler{}
	req := struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RedirectURI  string `json:"redirect_uri"`
	}{
		Code:         "valid-authorization-code",
		CodeVerifier: "short",
		RedirectURI:  "https://example.com/callback",
	}

	// Test
	err := handler.validateTokenExchangeInputs(req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code verifier too short")
}

func TestValidateTokenExchangeInputs_InvalidRedirectURI(t *testing.T) {
	// Setup
	handler := &handlers.AuthHandler{}
	req := struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RedirectURI  string `json:"redirect_uri"`
	}{
		Code:         "valid-authorization-code",
		CodeVerifier: "valid-code-verifier-with-sufficient-length-for-pkce",
		RedirectURI:  "http://example.com/callback", // HTTP instead of HTTPS
	}

	// Test
	err := handler.validateTokenExchangeInputs(req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must use HTTPS")
}

func TestValidateAuthentikToken_StandardJWT(t *testing.T) {
	// Setup - mock a standard 3-segment JWT
	handler := &handlers.AuthHandler{}
	tokenResp := &authentikTokenResponse{
		IDToken:      "header.payload.signature",
		AccessToken: "",
	}

	// Test - this should pass basic format validation
	// Note: Full OIDC validation would require mocking the OIDC service
	// This test validates the format checking logic
	err := handler.validateAuthentikToken(tokenResp)

	// We expect this to fail at OIDC validation since we don't have a real service
	// but it should not fail due to segment count validation
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC token validation failed")
	assert.NotContains(t, err.Error(), "invalid JWT structure")
	assert.NotContains(t, err.Error(), "expected 3")
}

func TestValidateAuthentikToken_MultiSegmentToken(t *testing.T) {
	// Setup - mock a token with more than 3 segments (simulates the user's issue)
	handler := &handlers.AuthHandler{}
	tokenResp := &authentikTokenResponse{
		IDToken:      "header.part1.part2.part3.part4.part5", // 5 segments like the user's error
		AccessToken: "",
	}

	// Test - this should attempt reconstruction and not fail on segment count
	err := handler.validateAuthentikToken(tokenResp)

	// Should fail at OIDC validation but not segment validation
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC token validation failed")
	assert.Contains(t, err.Error(), "original segments: 5")
	assert.Contains(t, err.Error(), "reconstructed segments: 3")
}

func TestValidateAuthentikToken_NoToken(t *testing.T) {
	// Setup - no tokens provided
	handler := &handlers.AuthHandler{}
	tokenResp := &authentikTokenResponse{
		IDToken:     "",
		AccessToken: "",
	}

	// Test
	err := handler.validateAuthentikToken(tokenResp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid token received from authentik")
}

func TestValidateAuthentikToken_NoDots(t *testing.T) {
	// Setup - token without dots (not a JWT)
	handler := &handlers.AuthHandler{}
	tokenResp := &authentikTokenResponse{
		IDToken: "not-a-jwt-token",
	}

	// Test
	err := handler.validateAuthentikToken(tokenResp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid JWT format")
	assert.Contains(t, err.Error(), "no dots found")
}

// Note: Integration tests for full token exchange flow would require:
// - Mock HTTP server for Authentik
// - Test database setup
// - Mock OIDC service
// These unit tests focus on validation logic only
