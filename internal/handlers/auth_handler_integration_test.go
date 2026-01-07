//go:build integration

package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestValidateAuthentikToken_UserInfoSuccess(t *testing.T) {
	oidcSvc, cleanup := newTestOIDCService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userinfo" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"email":              "alice@example.com",
				"preferred_username": "alice",
				"name":               "Alice",
				"groups":             []string{"admins"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer cleanup()

	h := &AuthHandler{oidcService: oidcSvc}
	claims, err := h.validateAuthentikToken(&authentikTokenResponse{AccessToken: "test-access-token"})
	assert.NoError(t, err)
	assert.Equal(t, "alice@example.com", claims.Email)
	assert.Equal(t, "alice", claims.PreferredUsername)
	assert.Equal(t, "Alice", claims.Name)
	assert.Contains(t, claims.Groups, "admins")
}

func TestValidateAuthentikToken_UserInfoMissingEmail(t *testing.T) {
	oidcSvc, cleanup := newTestOIDCService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userinfo" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"preferred_username": "alice",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer cleanup()

	h := &AuthHandler{oidcService: oidcSvc}
	_, err := h.validateAuthentikToken(&authentikTokenResponse{AccessToken: "test-access-token"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate token via userinfo")
	assert.Contains(t, err.Error(), "email claim missing")
}

func newTestOIDCService(t *testing.T, userInfoHandler http.Handler) (*services.OIDCService, func()) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())

	mux := http.NewServeMux()
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": "test",
					"use": "sig",
					"n":   n,
					"e":   e,
				},
			},
		})
	})
	mux.Handle("/userinfo", userInfoHandler)

	srv := httptest.NewServer(mux)

	cfg := &config.Config{
		AuthentikIssuer:      "https://auth.example.com",
		AuthentikJWKSURL:     srv.URL + "/jwks",
		AuthentikClientID:    "test-client-id",
		AuthentikUserInfoURL: srv.URL + "/userinfo",
	}

	oidcSvc, err := services.NewOIDCService(cfg)
	if err != nil {
		srv.Close()
		t.Fatalf("failed to create oidc service: %v", err)
	}

	return oidcSvc, srv.Close
}
