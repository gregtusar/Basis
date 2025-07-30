package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gregtusar/basis/pkg/models"
	"github.com/gregtusar/basis/pkg/trader"
	"github.com/sirupsen/logrus"
)

type Server struct {
	trader *trader.BasisTrader
	logger *logrus.Logger
	port   string
}

func NewServer(trader *trader.BasisTrader, logger *logrus.Logger, port string) *Server {
	return &Server{
		trader: trader,
		logger: logger,
		port:   port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// API endpoints
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/basis/snapshots", s.handleBasisSnapshots)
	mux.HandleFunc("/api/strategies", s.handleStrategies)
	mux.HandleFunc("/api/positions", s.handlePositions)
	mux.HandleFunc("/api/trades", s.handleTrades)
	
	// Enable CORS for Streamlit
	handler := corsMiddleware(mux)
	
	s.logger.Infof("Starting API server on port %s", s.port)
	return http.ListenAndServe(":"+s.port, handler)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
	}
	
	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleBasisSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	snapshots := s.trader.GetBasisSnapshots()
	s.writeJSON(w, http.StatusOK, snapshots)
}

func (s *Server) handleStrategies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// TODO: Implement get strategies
		s.writeJSON(w, http.StatusOK, []models.BasisStrategy{})
		
	case http.MethodPost:
		var strategy models.BasisStrategy
		if err := json.NewDecoder(r.Body).Decode(&strategy); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		strategy.ID = generateID()
		strategy.CreatedAt = time.Now()
		strategy.UpdatedAt = time.Now()
		
		if err := s.trader.AddStrategy(&strategy); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		s.writeJSON(w, http.StatusCreated, strategy)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePositions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// TODO: Implement get positions from trader
	s.writeJSON(w, http.StatusOK, []models.Position{})
}

func (s *Server) handleTrades(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// TODO: Implement get trades history
	s.writeJSON(w, http.StatusOK, []models.BasisTrade{})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func generateID() string {
	return time.Now().Format("20060102150405")
}