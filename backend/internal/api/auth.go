package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// authMiddleware valida o header Authorization: Bearer <token> contra o token
// configurado. Se o token env estiver vazio (dev local), o middleware é no-op.
// /api/health é sempre público (liveness probe).
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dev local sem token configurado: auth desligada.
		if s.APIToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Health endpoint é público.
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}

		provided := r.Header.Get("Authorization")
		if !strings.HasPrefix(provided, "Bearer ") {
			respondError(w, http.StatusUnauthorized, errMissingToken)
			return
		}
		token := strings.TrimPrefix(provided, "Bearer ")

		// subtle.ConstantTimeCompare evita timing attack na comparação do token.
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.APIToken)) != 1 {
			respondError(w, http.StatusUnauthorized, errInvalidToken)
			return
		}

		// Anexa o request ID do chi no contexto pra correlacionar logs.
		ctx := context.WithValue(r.Context(), requestIDKey{}, middleware.GetReqID(r.Context()))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type requestIDKey struct{}

var (
	errMissingToken = &authError{"token de autenticação ausente"}
	errInvalidToken = &authError{"token de autenticação inválido"}
)

type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }
