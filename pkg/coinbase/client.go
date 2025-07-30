package coinbase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gregtusar/basis/pkg/models"
)

type Client interface {
	GetTicker(ctx context.Context, symbol string) (*models.Ticker, error)
	GetOrderBook(ctx context.Context, symbol string, level int) (*models.OrderBook, error)
	GetPositions(ctx context.Context) ([]models.Position, error)
	PlaceOrder(ctx context.Context, order *models.OrderRequest) (*models.Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (*models.Order, error)
	Subscribe(channels []string, symbols []string) error
}

type BaseClient struct {
	auth       Authenticator
	baseURL    string
	httpClient *http.Client
}

type AdvancedTradeClient struct {
	BaseClient
}

type PrimeClient struct {
	BaseClient
}

// NewAdvancedTradeClient creates a client with legacy authentication (for backward compatibility)
func NewAdvancedTradeClient(apiKey, apiSecret, passphrase string, sandbox bool) *AdvancedTradeClient {
	baseURL := "https://api.coinbase.com"
	if sandbox {
		baseURL = "https://api-public.sandbox.coinbase.com"
	}

	return &AdvancedTradeClient{
		BaseClient: BaseClient{
			auth:       NewLegacyAuthenticator(apiKey, apiSecret, passphrase),
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}
}

// NewAdvancedTradeClientJWT creates a client with JWT authentication
func NewAdvancedTradeClientJWT(apiKeyName, privateKeyPEM string, sandbox bool) (*AdvancedTradeClient, error) {
	baseURL := "https://api.coinbase.com"
	if sandbox {
		baseURL = "https://api-public.sandbox.coinbase.com"
	}

	auth, err := NewJWTAuthenticator(apiKeyName, privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT authenticator: %w", err)
	}

	return &AdvancedTradeClient{
		BaseClient: BaseClient{
			auth:       auth,
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}, nil
}

// NewPrimeClient creates a client with legacy authentication (Prime still uses this)
func NewPrimeClient(apiKey, apiSecret, passphrase string, sandbox bool) *PrimeClient {
	baseURL := "https://api.prime.coinbase.com"
	if sandbox {
		baseURL = "https://api-public.sandbox.prime.coinbase.com"
	}

	return &PrimeClient{
		BaseClient: BaseClient{
			auth:       NewLegacyAuthenticator(apiKey, apiSecret, passphrase),
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}
}

// computeHMAC calculates HMAC for legacy authentication
func computeHMAC(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (c *BaseClient) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication headers
	if err := c.auth.AddAuthHeaders(req, method, path, string(body)); err != nil {
		return nil, fmt.Errorf("failed to add auth headers: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}