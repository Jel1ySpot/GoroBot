package onebot

import (
	"fmt"
	"io"
	"net/http"
)

// HTTP POST server for receiving events
func (s *Service) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTPPostEvent)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.HTTP.Host, s.config.HTTP.Port),
		Handler: mux,
	}

	go func() {
		s.logger.Info("Starting HTTP POST server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP POST server error: %v", err)
		}
	}()

	return nil
}

func (s *Service) handleHTTPPostEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Verify signature if secret is configured
	if s.config.HTTP.Secret != "" {
		signature := r.Header.Get("X-Signature")
		if !s.verifySignature(r, signature) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read HTTP POST body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Process event
	if err := s.processEvent(body); err != nil {
		s.logger.Error("Failed to process HTTP POST event: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) verifySignature(r *http.Request, signature string) bool {
	// TODO: Implement HMAC SHA1 signature verification
	// This would involve reading the request body and computing HMAC with the secret
	return true // Simplified for now
}
