package secrets

import (
	"context"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/sirupsen/logrus"
)

type GCPSecretManager struct {
	client    *secretmanager.Client
	projectID string
	logger    *logrus.Logger
}

func NewGCPSecretManager(ctx context.Context, projectID string, logger *logrus.Logger) (*GCPSecretManager, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %w", err)
	}

	return &GCPSecretManager{
		client:    client,
		projectID: projectID,
		logger:    logger,
	}, nil
}

func (g *GCPSecretManager) GetSecret(ctx context.Context, secretName string) (string, error) {
	// Build the resource name of the secret version
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", g.projectID, secretName)

	// Access the secret version
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := g.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret %s: %w", secretName, err)
	}

	// Extract the payload
	data := result.Payload.Data
	return string(data), nil
}

func (g *GCPSecretManager) GetSecretWithDefault(ctx context.Context, secretName, defaultValue string) string {
	value, err := g.GetSecret(ctx, secretName)
	if err != nil {
		g.logger.WithError(err).WithField("secret", secretName).Debug("Failed to get secret, using default")
		return defaultValue
	}
	return strings.TrimSpace(value)
}

func (g *GCPSecretManager) Close() error {
	return g.client.Close()
}

type SecretNames struct {
	// Spot trading secrets
	SpotAPIKey       string
	SpotAPISecret    string
	SpotPassphrase   string
	
	// Derivatives trading secrets
	DerivativesAPIKey       string
	DerivativesAPISecret    string
	DerivativesPassphrase   string
}

func DefaultSecretNames() SecretNames {
	return SecretNames{
		SpotAPIKey:              "coinbase-spot-api-key",
		SpotAPISecret:           "coinbase-spot-api-secret",
		SpotPassphrase:          "coinbase-spot-passphrase",
		DerivativesAPIKey:       "coinbase-derivatives-api-key",
		DerivativesAPISecret:    "coinbase-derivatives-api-secret",
		DerivativesPassphrase:   "coinbase-derivatives-passphrase",
	}
}