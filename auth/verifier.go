// Package auth provides OIDC token verification for PolyON modules.
//
// It fetches JWKS from the internal Keycloak endpoint and validates Bearer tokens.
// Use Middleware() to protect HTTP handlers, or VerifyToken() for manual verification.
package auth

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

	polyon "github.com/jupiter-ai-agent/PolyON-sdk-go"
)

// Claims represents decoded JWT claims from a PolyON OIDC token.
type Claims struct {
	Sub               string   `json:"sub"`
	PreferredUsername  string   `json:"preferred_username"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	GivenName         string   `json:"given_name"`
	FamilyName        string   `json:"family_name"`
	RealmAccess       Roles    `json:"realm_access"`
	ResourceAccess    map[string]Roles `json:"resource_access"`
	Issuer            string   `json:"iss"`
	Audience          any      `json:"aud"` // string or []string
	ExpiresAt         int64    `json:"exp"`
	IssuedAt          int64    `json:"iat"`
}

// Roles holds Keycloak role assignments.
type Roles struct {
	Roles []string `json:"roles"`
}

// HasRole checks if the user has a specific realm role.
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.RealmAccess.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasClientRole checks if the user has a specific client role.
func (c *Claims) HasClientRole(client, role string) bool {
	if ra, ok := c.ResourceAccess[client]; ok {
		for _, r := range ra.Roles {
			if r == role {
				return true
			}
		}
	}
	return false
}

// Verifier validates OIDC JWTs using JWKS from Keycloak.
type Verifier struct {
	issuer   string
	clientID string
	jwksURI  string

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// NewVerifier creates a Verifier from PRC auth config.
func NewVerifier(cfg polyon.AuthConfig) *Verifier {
	return &Verifier{
		issuer:   cfg.Issuer,
		clientID: cfg.ClientID,
		jwksURI:  cfg.JWKSURI, // internal URL (polyon-auth:8080)
		keys:     map[string]*rsa.PublicKey{},
	}
}

// Middleware returns an HTTP middleware that validates Bearer tokens.
// On success, Claims are stored in the request context.
// On failure, responds with 401 Unauthorized.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		claims, err := v.VerifyToken(r.Context(), token)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// VerifyToken validates a JWT string and returns its claims.
func (v *Verifier) VerifyToken(ctx context.Context, tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// Decode header
	headerBytes, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	// Get public key
	key, err := v.getKey(ctx, header.Kid)
	if err != nil {
		return nil, fmt.Errorf("get signing key: %w", err)
	}

	// Verify signature
	if err := verifyRS256(parts[0]+"."+parts[1], parts[2], key); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Decode claims
	claimsBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	// Validate issuer
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("issuer mismatch: got %s, want %s", claims.Issuer, v.issuer)
	}

	// Validate expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

// getKey returns the RSA public key for a given kid, fetching JWKS if needed.
func (v *Verifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	v.mu.RUnlock()
	if ok {
		return key, nil
	}

	// Fetch JWKS (cache for 5 minutes)
	if err := v.fetchJWKS(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %s not found in JWKS", kid)
	}
	return key, nil
}

// fetchJWKS downloads the JWKS from Keycloak's internal endpoint.
func (v *Verifier) fetchJWKS(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Rate limit: don't re-fetch within 30 seconds
	if time.Since(v.fetched) < 30*time.Second {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", v.jwksURI, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("JWKS fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("JWKS fetch returned %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("JWKS decode: %w", err)
	}

	newKeys := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64URLDecode(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64URLDecode(k.E)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		newKeys[k.Kid] = &rsa.PublicKey{N: n, E: e}
	}

	v.keys = newKeys
	v.fetched = time.Now()
	return nil
}

// ── Context helpers ──

type claimsKey struct{}

// GetClaims retrieves the verified Claims from a request context.
// Returns nil if the request was not authenticated.
func GetClaims(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey{}).(*Claims)
	return claims
}

// ── Helpers ──

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
