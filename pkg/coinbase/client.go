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
	apiKey     string
	apiSecret  string
	passphrase string
	baseURL    string
	httpClient *http.Client
}

type AdvancedTradeClient struct {
	BaseClient
}

type PrimeClient struct {
	BaseClient
}

func NewAdvancedTradeClient(apiKey, apiSecret, passphrase string, sandbox bool) *AdvancedTradeClient {
	baseURL := "https://api.coinbase.com"
	if sandbox {
		baseURL = "https://api-public.sandbox.coinbase.com"
	}

	return &AdvancedTradeClient{
		BaseClient: BaseClient{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			passphrase: passphrase,
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}
}

func NewPrimeClient(apiKey, apiSecret, passphrase string, sandbox bool) *PrimeClient {
	baseURL := "https://api.prime.coinbase.com"
	if sandbox {
		baseURL = "https://api-public.sandbox.prime.coinbase.com"
	}

	return &PrimeClient{
		BaseClient: BaseClient{
			apiKey:     apiKey,
			apiSecret:  apiSecret,
			passphrase: passphrase,
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		},
	}
}

func (c *BaseClient) sign(method, path, body string, timestamp string) string {
	message := timestamp + method + path + body
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (c *BaseClient) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	signature := c.sign(method, path, string(body), timestamp)
	
	req.Header.Set("CB-ACCESS-KEY", c.apiKey)
	req.Header.Set("CB-ACCESS-SIGN", signature)
	req.Header.Set("CB-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("CB-ACCESS-PASSPHRASE", c.passphrase)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}