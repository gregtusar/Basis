# Build stage for Go application
FROM golang:1.21-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates python3 py3-pip

# Create non-root user
RUN addgroup -g 1001 -S trader && \
    adduser -u 1001 -S trader -G trader

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/bin/basis-trader /app/basis-trader

# Copy configuration files
COPY config.yaml /app/config.yaml

# Copy Streamlit files
COPY streamlit/ /app/streamlit/

# Install Python dependencies
RUN pip3 install --no-cache-dir -r /app/streamlit/requirements.txt

# Create data directory
RUN mkdir -p /app/data && chown -R trader:trader /app

# Switch to non-root user
USER trader

# Expose ports
EXPOSE 8080 8501

# Create startup script
RUN echo '#!/bin/sh\n\
/app/basis-trader &\n\
cd /app/streamlit && streamlit run app.py --server.address 0.0.0.0\n\
' > /app/start.sh && chmod +x /app/start.sh

# Run both services
CMD ["/app/start.sh"]