package coinbase

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthType represents the authentication method
type AuthType string

const (
	AuthTypeLegacy AuthType = "legacy"
	AuthTypeJWT    AuthType = "jwt"
)

// Authenticator interface for different auth methods
type Authenticator interface {
	AddAuthHeaders(req *http.Request, method, path, body string) error
}

// LegacyAuthenticator uses the traditional API Key/Secret/Passphrase
type LegacyAuthenticator struct {
	apiKey     string
	apiSecret  string
	passphrase string
}

func NewLegacyAuthenticator(apiKey, apiSecret, passphrase string) *LegacyAuthenticator {
	return &LegacyAuthenticator{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		passphrase: passphrase,
	}
}

func (l *LegacyAuthenticator) AddAuthHeaders(req *http.Request, method, path, body string) error {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := l.sign(method, path, body, timestamp)
	
	req.Header.Set("CB-ACCESS-KEY", l.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", l.passphrase)
	
	return nil
}

func (l *LegacyAuthenticator) sign(method, path, body, timestamp string) string {
	message := timestamp + method + path + body
	return computeHMAC(message, l.apiSecret)
}

// JWTAuthenticator uses the new JWT-based authentication
type JWTAuthenticator struct {
	apiKeyName string
	privateKey *ecdsa.PrivateKey
}

func NewJWTAuthenticator(apiKeyName, privateKeyPEM string) (*JWTAuthenticator, error) {
	// Parse the private key
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the private key")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse EC private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an EC private key")
		}
	}

	return &JWTAuthenticator{
		apiKeyName: apiKeyName,
		privateKey: privateKey,
	}, nil
}

func (j *JWTAuthenticator) AddAuthHeaders(req *http.Request, method, path, body string) error {
	token, err := j.generateJWT(method, req.Host, path)
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (j *JWTAuthenticator) generateJWT(method, host, path string) (string, error) {
	// Generate nonce
	nonce, err := generateNonce()
	if err != nil {
		return "", err
	}

	// JWT claims
	claims := jwt.MapClaims{
		"sub": j.apiKeyName,
		"iss": "coinbase-cloud",
		"nbf": time.Now().Unix(),
		"exp": time.Now().Add(2 * time.Minute).Unix(),
		"uri": method + " " + host + path,
		"nonce": nonce,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = j.apiKeyName
	token.Header["nonce"] = nonce

	// Sign token
	tokenString, err := token.SignedString(j.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// parseAPIKeyName extracts the org ID and key ID from the API key name
func parseAPIKeyName(apiKeyName string) (orgID, keyID string, err error) {
	// Expected format: organizations/{org_id}/apiKeys/{key_id}
	parts := strings.Split(apiKeyName, "/")
	if len(parts) != 4 || parts[0] != "organizations" || parts[2] != "apiKeys" {
		return "", "", fmt.Errorf("invalid API key name format")
	}
	return parts[1], parts[3], nil
}