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

// Note: Integration tests for full token exchange flow would require:
// - Mock HTTP server for Authentik
// - Test database setup
// - Mock OIDC service
// These unit tests focus on validation logic only
