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

// SupabaseClaims represents the standard claims from a Supabase JWT
type SupabaseClaims struct {
	Email string `json:"email"`
	Sub   string `json:"sub"`
	jwt.RegisteredClaims
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

// SupabaseAuthService handles validation of Supabase tokens via JWKS (RS256)
type SupabaseAuthService struct {
	cfg         *config.Config
	httpClient  *http.Client
	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	lastRefresh time.Time
	jwksURL     string
}

func NewSupabaseAuthService(cfg *config.Config) (*SupabaseAuthService, error) {
	jwksURL := cfg.SupabaseJWKSURL
	if jwksURL == "" {
		if cfg.SupabaseURL != "" {
			// Derive JWKS URL from Supabase URL (e.g. https://xyz.supabase.co -> https://xyz.supabase.co/auth/v1/jwks)
			baseUrl := strings.TrimSuffix(cfg.SupabaseURL, "/")
			jwksURL = fmt.Sprintf("%s/auth/v1/jwks", baseUrl)
		} else {
			return nil, errors.New("supabase configuration is incomplete - SUPABASE_URL or SUPABASE_JWKS_URL is required")
		}
	}

	svc := &SupabaseAuthService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
		jwksURL:    jwksURL,
	}

	// Initial key fetch
	if err := svc.refreshKeys(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to fetch Supabase JWKS from %s: %w", jwksURL, err)
	}

	return svc, nil
}

func (s *SupabaseAuthService) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.jwksURL, nil)
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
		return errors.New("no valid RSA keys found in Supabase JWKS")
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

func (s *SupabaseAuthService) keyForToken(token *jwt.Token) (*rsa.PublicKey, error) {
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

// ValidateToken validates an incoming Supabase JWT token and returns the claims
func (s *SupabaseAuthService) ValidateToken(tokenString string) (*SupabaseClaims, error) {
	claims := &SupabaseClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.keyForToken(token)
	}, jwt.WithValidMethods([]string{"RS256"}))

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid Supabase token")
	}

	if claims.Email == "" {
		return nil, errors.New("email claim missing from Supabase token")
	}

	return claims, nil
}
