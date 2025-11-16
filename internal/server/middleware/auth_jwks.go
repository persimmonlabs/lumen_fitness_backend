package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSKeyFunc creates a key function that validates JWTs using JWKS (JSON Web Key Set)
// This supports Supabase's new JWT signing key rotation feature
type JWKSKeyFunc struct {
	jwksURL    string
	keys       map[string]*rsa.PublicKey
	lastUpdate time.Time
	mu         sync.RWMutex
	httpClient *http.Client
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"` // Key ID
	Kty string `json:"kty"` // Key Type (RSA, EC, etc.)
	Alg string `json:"alg"` // Algorithm
	Use string `json:"use"` // Public key use (sig, enc)
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// NewJWKSKeyFunc creates a new JWKS key function
func NewJWKSKeyFunc(supabaseURL string) *JWKSKeyFunc {
	return &JWKSKeyFunc{
		jwksURL: fmt.Sprintf("%s/auth/v1/jwks", supabaseURL),
		keys:    make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetKey returns the key function for JWT validation
func (j *JWKSKeyFunc) GetKey(token *jwt.Token) (interface{}, error) {
	// Get the key ID from token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		// No kid in header - this is a legacy token, fallback to HS256
		return nil, fmt.Errorf("no kid in token header (legacy token)")
	}

	// Check if we need to refresh keys (every 5 minutes)
	j.mu.RLock()
	needsRefresh := time.Since(j.lastUpdate) > 5*time.Minute
	j.mu.RUnlock()

	if needsRefresh {
		if err := j.refreshKeys(); err != nil {
			return nil, fmt.Errorf("failed to refresh JWKS keys: %w", err)
		}
	}

	// Get the key
	j.mu.RLock()
	key, exists := j.keys[kid]
	j.mu.RUnlock()

	if !exists {
		// Try refreshing one more time in case it's a new key
		if err := j.refreshKeys(); err != nil {
			return nil, fmt.Errorf("failed to refresh JWKS keys: %w", err)
		}

		j.mu.RLock()
		key, exists = j.keys[kid]
		j.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
		}
	}

	return key, nil
}

// refreshKeys fetches the latest JWKS from Supabase
func (j *JWKSKeyFunc) refreshKeys() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", j.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert JWKs to RSA public keys
	newKeys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			// Skip non-RSA keys for now
			continue
		}

		publicKey, err := j.parseRSAKey(jwk)
		if err != nil {
			// Log but don't fail - just skip this key
			continue
		}

		newKeys[jwk.Kid] = publicKey
	}

	// Update keys
	j.mu.Lock()
	j.keys = newKeys
	j.lastUpdate = time.Now()
	j.mu.Unlock()

	return nil
}

// parseRSAKey converts a JWK to an RSA public key
func (j *JWKSKeyFunc) parseRSAKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode the modulus (n)
	nBytes, err := base64URLDecode(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent (e)
	eBytes, err := base64URLDecode(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// Create RSA public key
	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return publicKey, nil
}

// base64URLDecode decodes a base64 URL-encoded string
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// JWT uses URL-safe base64 encoding, convert to standard encoding
	s = strings.Replace(s, "-", "+", -1)
	s = strings.Replace(s, "_", "/", -1)

	// Decode using standard base64
	return base64.StdEncoding.DecodeString(s)
}

// validateJWTWithJWKS validates a JWT token using JWKS
func validateJWTWithJWKS(tokenString, supabaseURL, fallbackSecret string) (string, error) {
	// Create JWKS key function
	jwksKeyFunc := NewJWKSKeyFunc(supabaseURL)

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Try JWKS first (for new tokens with kid)
		if _, ok := token.Header["kid"]; ok {
			return jwksKeyFunc.GetKey(token)
		}

		// Fallback to HS256 for legacy tokens without kid
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(fallbackSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Extract user ID from "sub" claim
	sub, ok := claims["sub"]
	if !ok {
		return "", fmt.Errorf("missing sub claim")
	}

	userID, ok := sub.(string)
	if !ok {
		return "", fmt.Errorf("invalid sub claim type")
	}

	if userID == "" {
		return "", fmt.Errorf("empty user ID")
	}

	return userID, nil
}
