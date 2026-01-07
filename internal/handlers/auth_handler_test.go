package handlers

import (
	"testing"

	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestValidateAuthentikConfig_Success(t *testing.T) {
	cfg := &config.Config{
		AuthentikEnabled:     true,
		AuthentikTokenURL:    "https://auth.example.com/token",
		AuthentikUserInfoURL: "https://auth.example.com/userinfo",
		AuthentikClientID:    "test-client-id",
	}

	h := &AuthHandler{
		cfg:         cfg,
		oidcService: &services.OIDCService{}, // non-nil is sufficient for config validation
	}

	assert.NoError(t, h.validateAuthentikConfig())
}

func TestValidateAuthentikConfig_Disabled(t *testing.T) {
	cfg := &config.Config{AuthentikEnabled: false}
	h := &AuthHandler{cfg: cfg}

	err := h.validateAuthentikConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestValidateAuthentikConfig_NoOIDCService(t *testing.T) {
	cfg := &config.Config{
		AuthentikEnabled:     true,
		AuthentikTokenURL:    "https://auth.example.com/token",
		AuthentikUserInfoURL: "https://auth.example.com/userinfo",
		AuthentikClientID:    "test-client-id",
	}
	h := &AuthHandler{cfg: cfg}

	err := h.validateAuthentikConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OIDC service not initialized")
}

func TestValidateTokenExchangeInputs_Valid(t *testing.T) {
	h := &AuthHandler{}
	req := struct {
		Code         string `json:"code" binding:"required"`
		CodeVerifier string `json:"code_verifier" binding:"required"`
		RedirectURI  string `json:"redirect_uri" binding:"required"`
	}{
		Code:         "valid-authorization-code",
		CodeVerifier: "this-code-verifier-is-long-enough-to-pass-the-43-char-minimum",
		RedirectURI:  "https://example.com/callback",
	}

	assert.NoError(t, h.validateTokenExchangeInputs(req))
}

func TestValidateTokenExchangeInputs_CodeTooShort(t *testing.T) {
	h := &AuthHandler{}
	req := struct {
		Code         string `json:"code" binding:"required"`
		CodeVerifier string `json:"code_verifier" binding:"required"`
		RedirectURI  string `json:"redirect_uri" binding:"required"`
	}{
		Code:         "short",
		CodeVerifier: "this-code-verifier-is-long-enough-to-pass-the-43-char-minimum",
		RedirectURI:  "https://example.com/callback",
	}

	err := h.validateTokenExchangeInputs(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authorization code too short")
}

func TestValidateTokenExchangeInputs_CodeVerifierTooShort(t *testing.T) {
	h := &AuthHandler{}
	req := struct {
		Code         string `json:"code" binding:"required"`
		CodeVerifier string `json:"code_verifier" binding:"required"`
		RedirectURI  string `json:"redirect_uri" binding:"required"`
	}{
		Code:         "valid-authorization-code",
		CodeVerifier: "short",
		RedirectURI:  "https://example.com/callback",
	}

	err := h.validateTokenExchangeInputs(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "code verifier too short")
}

func TestValidateTokenExchangeInputs_InvalidRedirectURI(t *testing.T) {
	h := &AuthHandler{}
	req := struct {
		Code         string `json:"code" binding:"required"`
		CodeVerifier string `json:"code_verifier" binding:"required"`
		RedirectURI  string `json:"redirect_uri" binding:"required"`
	}{
		Code:         "valid-authorization-code",
		CodeVerifier: "this-code-verifier-is-long-enough-to-pass-the-43-char-minimum",
		RedirectURI:  "http://example.com/callback", // HTTP instead of HTTPS
	}

	err := h.validateTokenExchangeInputs(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must use HTTPS")
}

func TestValidateAuthentikToken_NoAccessToken(t *testing.T) {
	h := &AuthHandler{oidcService: &services.OIDCService{}}
	_, err := h.validateAuthentikToken(&authentikTokenResponse{AccessToken: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no access token received from authentik")
}
