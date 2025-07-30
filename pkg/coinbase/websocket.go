package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WebSocketClient struct {
	url          string
	apiKey       string
	apiSecret    string
	passphrase   string
	conn         *websocket.Conn
	mu           sync.Mutex
	connected    bool
	subscriptions map[string]bool
	handlers     map[string]MessageHandler
	logger       *logrus.Logger
}

type MessageHandler func(message json.RawMessage) error

type WSMessage struct {
	Type      string          `json:"type"`
	ProductID string          `json:"product_id"`
	Time      time.Time       `json:"time"`
	Sequence  int64           `json:"sequence"`
	Message   json.RawMessage `json:"-"`
}

type SubscribeMessage struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids"`
	Channels   []string `json:"channels"`
	Signature  string   `json:"signature"`
	Key        string   `json:"key"`
	Passphrase string   `json:"passphrase"`
	Timestamp  string   `json:"timestamp"`
}

func NewWebSocketClient(url, apiKey, apiSecret, passphrase string, logger *logrus.Logger) *WebSocketClient {
	return &WebSocketClient{
		url:           url,
		apiKey:        apiKey,
		apiSecret:     apiSecret,
		passphrase:    passphrase,
		subscriptions: make(map[string]bool),
		handlers:      make(map[string]MessageHandler),
		logger:        logger,
	}
}

func (ws *WebSocketClient) Connect(ctx context.Context) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.connected {
		return nil
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, ws.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	ws.conn = conn
	ws.connected = true

	go ws.readLoop(ctx)
	go ws.keepAlive(ctx)

	return nil
}

func (ws *WebSocketClient) Subscribe(channels []string, productIDs []string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.connected {
		return fmt.Errorf("websocket not connected")
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	
	sub := SubscribeMessage{
		Type:       "subscribe",
		ProductIDs: productIDs,
		Channels:   channels,
		Key:        ws.apiKey,
		Passphrase: ws.passphrase,
		Timestamp:  timestamp,
	}

	// Generate signature
	message := timestamp + "GET" + "/users/self/verify"
	sub.Signature = ws.sign(message)

	return ws.conn.WriteJSON(sub)
}

func (ws *WebSocketClient) RegisterHandler(messageType string, handler MessageHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.handlers[messageType] = handler
}

func (ws *WebSocketClient) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg WSMessage
			err := ws.conn.ReadJSON(&msg)
			if err != nil {
				ws.logger.WithError(err).Error("Failed to read websocket message")
				ws.handleDisconnect()
				return
			}

			if handler, ok := ws.handlers[msg.Type]; ok {
				if err := handler(msg.Message); err != nil {
					ws.logger.WithError(err).Error("Handler error")
				}
			}
		}
	}
}

func (ws *WebSocketClient) keepAlive(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ws.mu.Lock()
			if ws.connected {
				if err := ws.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					ws.logger.WithError(err).Error("Failed to send ping")
					ws.handleDisconnect()
				}
			}
			ws.mu.Unlock()
		}
	}
}

func (ws *WebSocketClient) handleDisconnect() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	ws.connected = false
	if ws.conn != nil {
		ws.conn.Close()
	}
}

func (ws *WebSocketClient) sign(message string) string {
	// Implementation would be similar to BaseClient.sign
	return ""
}