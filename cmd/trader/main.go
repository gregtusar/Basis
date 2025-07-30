package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gregtusar/basis/api"
	"github.com/gregtusar/basis/internal/config"
	"github.com/gregtusar/basis/pkg/coinbase"
	"github.com/gregtusar/basis/pkg/trader"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	logger  *logrus.Logger
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "basis-trader",
		Short: "Cryptocurrency basis trading system",
		Long:  `A sophisticated trading system for executing basis trades between spot and perpetual futures markets`,
		Run:   runTrader,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runTrader(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}
	
	// Set log level
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logger.WithError(err).Error("Invalid log level, using INFO")
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Initialize Coinbase clients
	spotClient := coinbase.NewPrimeClient(
		cfg.Coinbase.Spot.APIKey,
		cfg.Coinbase.Spot.APISecret,
		cfg.Coinbase.Spot.Passphrase,
		cfg.Coinbase.Spot.Sandbox,
	)
	
	derivativesClient := coinbase.NewAdvancedTradeClient(
		cfg.Coinbase.Derivatives.APIKey,
		cfg.Coinbase.Derivatives.APISecret,
		cfg.Coinbase.Derivatives.Passphrase,
		cfg.Coinbase.Derivatives.Sandbox,
	)
	
	// Create basis trader
	basisTrader := trader.NewBasisTrader(spotClient, derivativesClient, logger)
	
	// Start the trader
	if err := basisTrader.Start(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to start basis trader")
	}
	
	// Start API server
	apiServer := api.NewServer(basisTrader, logger, fmt.Sprintf("%d", cfg.Server.Port))
	go func() {
		if err := apiServer.Start(); err != nil {
			logger.WithError(err).Fatal("Failed to start API server")
		}
	}()
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	logger.Info("Basis trader is running. Press Ctrl+C to stop.")
	
	<-sigChan
	logger.Info("Received shutdown signal")
	
	// Graceful shutdown
	basisTrader.Stop()
	cancel()
	
	logger.Info("Basis trader stopped")
}