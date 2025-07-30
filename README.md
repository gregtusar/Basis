# Basis Trading System

A sophisticated cryptocurrency basis trading system that executes arbitrage strategies between spot and perpetual futures markets using Coinbase APIs.

## Features

- **Dual Market Trading**: Simultaneously trades spot (via Coinbase Prime) and perpetual futures (via Advanced Trade API)
- **Real-time Market Data**: WebSocket connections for live price feeds and order book updates
- **Automated Strategy Execution**: Configurable basis trading strategies with automatic position management
- **Risk Management**: Built-in position limits, slippage controls, and rebalancing thresholds
- **Live Monitoring**: Streamlit-based dashboard for real-time strategy monitoring and control
- **RESTful API**: HTTP API for programmatic access and integration

## Architecture

```
├── cmd/trader/         # Main application entry point
├── pkg/
│   ├── coinbase/      # Coinbase API client implementations
│   ├── trader/        # Core trading logic and strategy execution
│   ├── models/        # Data structures for markets, orders, positions
│   └── utils/         # Utility functions
├── internal/
│   ├── config/        # Configuration management
│   └── storage/       # Data persistence layer
├── api/               # REST API server
├── streamlit/         # Monitoring dashboard
└── scripts/           # Utility scripts
```

## Prerequisites

- Go 1.21 or higher
- Python 3.8+ (for Streamlit dashboard)
- Coinbase API credentials:
  - Prime API credentials (for spot trading)
  - Advanced Trade API credentials (for derivatives)
- (Optional) Google Cloud Project with Secret Manager enabled

## Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/gregtusar/Basis.git
   cd Basis
   ```

2. **Set up environment**
   ```bash
   make setup
   ```

3. **Configure API credentials**
   
   **Option A: Using environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your Coinbase API credentials
   ```
   
   **Option B: Using GCP Secret Manager**
   ```bash
   # Set up GCP authentication
   gcloud auth application-default login
   
   # Create secrets in GCP
   echo -n "your-api-key" | gcloud secrets create coinbase-spot-api-key --data-file=-
   echo -n "your-api-secret" | gcloud secrets create coinbase-spot-api-secret --data-file=-
   # ... repeat for all secrets
   
   # Configure the application to use GCP
   export GCP_PROJECT_ID="your-project-id"
   export GCP_USE_SECRETS=true
   ```

4. **Run the system**
   ```bash
   # Run trader and dashboard
   make run-all
   
   # Or run separately:
   make run        # Start trader
   make streamlit  # Start dashboard
   ```

## Configuration

Edit `config.yaml` to customize:

- Trading parameters (position sizes, basis targets)
- API endpoints and connection settings
- Logging configuration
- Database location
- GCP Secret Manager settings

### Secret Management

The application supports two methods for managing API credentials:

1. **Environment Variables** (default): Store credentials in `.env` file or system environment
2. **GCP Secret Manager**: Store credentials securely in Google Cloud

To use GCP Secret Manager:
1. Enable the Secret Manager API in your GCP project
2. Create secrets with the appropriate names (see `config.yaml`)
3. Set `gcp.use_secrets: true` in config or `GCP_USE_SECRETS=true` in environment
4. Ensure the application has appropriate GCP credentials (via service account or ADC)

## API Endpoints

- `GET /api/health` - System health check
- `GET /api/basis/snapshots` - Current basis calculations
- `GET /api/strategies` - List active strategies
- `POST /api/strategies` - Create new strategy
- `GET /api/positions` - Current positions
- `GET /api/trades` - Trade history

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Build binary
make build

# Docker build
make docker-build
```

## Security

- Never commit API credentials
- Use GCP Secret Manager for production deployments
- Use environment variables only for development
- Enable sandbox mode for testing
- Implement proper rate limiting
- Ensure proper IAM permissions when using GCP

## License

[Your License Here]