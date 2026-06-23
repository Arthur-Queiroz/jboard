package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
)

// rotas sempre públicas (não exigem auth): liveness e o fluxo de login.
var publicPaths = map[string]bool{
	"/api/health": true,
	"/api/login":  true,
	"/api/logout": true,
}

// authMiddleware autoriza um request se ele tiver um Bearer válido (desktop/
// máquina) OU um cookie de sessão válido (web). Se o token env estiver vazio
// (dev local), o middleware é no-op.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.APIToken == "" { // dev local: auth desligada
			next.ServeHTTP(w, r)
			return
		}
		if publicPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		if s.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		respondError(w, http.StatusUnauthorized, errInvalidToken)
	})
}

// authorized aceita, em ordem: Bearer igual ao APIToken (máquina/scripts), Bearer
// com um token de sessão assinado (desktop, que é cross-origin e não usa cookie),
// ou o cookie de sessão (web, mesma origem).
func (s *Server) authorized(r *http.Request) bool {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token := strings.TrimPrefix(h, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.APIToken)) == 1 {
			return true
		}
		if validSession(s.APIToken, token, time.Now()) {
			return true
		}
	}
	if c, err := r.Cookie(sessionCookieName); err == nil {
		if validSession(s.APIToken, c.Value, time.Now()) {
			return true
		}
	}
	return false
}

// login troca a senha por um cookie de sessão httpOnly (web). O desktop não usa
// isto — manda Bearer direto.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password  string `json:"password"`
		WantToken bool   `json:"want_token"` // desktop (cross-origin) pede o token p/ Bearer
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	// Sem senha configurada ou senha errada → 401 (comparação constant-time).
	if s.AuthPassword == "" || subtle.ConstantTimeCompare([]byte(body.Password), []byte(s.AuthPassword)) != 1 {
		respondError(w, http.StatusUnauthorized, errInvalidToken)
		return
	}
	session := mintSession(s.APIToken, time.Now())
	s.setSessionCookie(w, session, int(sessionTTL.Seconds()))
	resp := map[string]string{"status": "ok"}
	if body.WantToken {
		// Só devolve o token quando pedido (desktop), pra não expor no JS da web.
		resp["token"] = session
	}
	respondJSON(w, http.StatusOK, resp)
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	s.setSessionCookie(w, "", -1) // expira o cookie
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // prod é HTTPS (Cloudflare); em dev a auth fica desligada
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

var errInvalidToken = &authError{"credencial de autenticação inválida"}

type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }
