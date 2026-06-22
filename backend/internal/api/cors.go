package api

import (
	"net/http"
	"strings"
)

// corsMiddleware libera acesso cross-origin pras origens da allowlist. O único
// cliente que precisa disso é o desktop Tauri empacotado, cujo webview tem origem
// própria (tauri://localhost); web e desktop-em-dev falam com a API na mesma
// origem (proxy do Vite / Caddy). A API usa token Bearer, não cookies, então não
// há credencial de origem a proteger — ainda assim restringimos a uma allowlist
// em vez de "*" por higiene.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "300")
		}
		// Preflight: responde já, antes da auth e dos handlers (o OPTIONS não leva
		// o header Authorization, então não deve passar pelo authMiddleware).
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin string) bool {
	for _, o := range s.AllowedOrigins {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}
