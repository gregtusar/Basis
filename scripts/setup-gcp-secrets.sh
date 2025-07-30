#!/bin/bash

# Script to set up GCP secrets for the Basis Trading System

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}Error: gcloud CLI is not installed${NC}"
    echo "Please install the Google Cloud SDK: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Get project ID
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
if [ -z "$PROJECT_ID" ]; then
    echo -e "${YELLOW}No default project set${NC}"
    read -p "Enter your GCP project ID: " PROJECT_ID
    gcloud config set project "$PROJECT_ID"
fi

echo -e "${GREEN}Using project: $PROJECT_ID${NC}"

# Enable Secret Manager API
echo "Enabling Secret Manager API..."
gcloud services enable secretmanager.googleapis.com --project="$PROJECT_ID"

# Function to create or update a secret
create_secret() {
    local secret_name=$1
    local secret_desc=$2
    
    echo -e "\n${YELLOW}Setting up secret: $secret_name${NC}"
    echo "Description: $secret_desc"
    
    # Check if secret exists
    if gcloud secrets describe "$secret_name" --project="$PROJECT_ID" &>/dev/null; then
        echo "Secret already exists. Do you want to update it? (y/n)"
        read -r response
        if [[ "$response" != "y" ]]; then
            echo "Skipping $secret_name"
            return
        fi
    else
        # Create the secret
        gcloud secrets create "$secret_name" \
            --project="$PROJECT_ID" \
            --replication-policy="automatic" \
            --labels="app=basis-trader"
    fi
    
    # Get the secret value
    echo -n "Enter value for $secret_name (input hidden): "
    read -s secret_value
    echo
    
    # Add secret version
    echo -n "$secret_value" | gcloud secrets versions add "$secret_name" \
        --project="$PROJECT_ID" \
        --data-file=-
    
    echo -e "${GREEN}✓ $secret_name configured${NC}"
}

# Create all required secrets
echo -e "\n${GREEN}Setting up Coinbase API secrets...${NC}"

create_secret "coinbase-spot-api-key" "Coinbase Prime API Key for spot trading"
create_secret "coinbase-spot-api-secret" "Coinbase Prime API Secret for spot trading"
create_secret "coinbase-spot-passphrase" "Coinbase Prime API Passphrase for spot trading"

create_secret "coinbase-derivatives-api-key" "Coinbase Advanced Trade API Key for derivatives"
create_secret "coinbase-derivatives-api-secret" "Coinbase Advanced Trade API Secret for derivatives"
create_secret "coinbase-derivatives-passphrase" "Coinbase Advanced Trade API Passphrase for derivatives"

# Grant permissions to the default service account (or user)
echo -e "\n${GREEN}Setting up permissions...${NC}"

# Get the current user or service account
CURRENT_IDENTITY=$(gcloud auth list --filter=status:ACTIVE --format="value(account)")
echo "Granting Secret Manager access to: $CURRENT_IDENTITY"

# Grant secretAccessor role
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member="user:$CURRENT_IDENTITY" \
    --role="roles/secretmanager.secretAccessor" \
    --condition=None

echo -e "\n${GREEN}Setup complete!${NC}"
echo -e "\nTo use GCP secrets in your application:"
echo "1. Set the following environment variables:"
echo "   export GCP_PROJECT_ID=$PROJECT_ID"
echo "   export GCP_USE_SECRETS=true"
echo ""
echo "2. Or update config.yaml:"
echo "   gcp:"
echo "     use_secrets: true"
echo "     project_id: $PROJECT_ID"
echo ""
echo "3. Make sure you're authenticated:"
echo "   gcloud auth application-default login"