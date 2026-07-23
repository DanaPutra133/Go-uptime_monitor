package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type ServerHandler struct {
	cfg *Config
}

func NewServerHandler(cfg *Config) *ServerHandler {
	return &ServerHandler{cfg: cfg}
}

func (sh *ServerHandler) SetupRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", sh.HandleRoot)

	return sh.MethodBlockerMiddleware(mux)
}

func (sh *ServerHandler) MethodBlockerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Method not allowed. Hanya method GET yang diizinkan."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (sh *ServerHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":          "online",
		"system":          "Uptime Monitor Service (Go)",
		"active_services": len(sh.cfg.Services),
		"checked_at":      time.Now().Format("2006-01-02 15:04:05"),
		// "services_list":   sh.cfg.Services,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}