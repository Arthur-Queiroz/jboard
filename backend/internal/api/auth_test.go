package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// authServer monta um Server com auth ligada (APIToken não-vazio).
func authServer() http.Handler {
	store := newFakeStore()
	return (&Server{
		Boards:    store,
		Columns:   store,
		Cards:     store,
		Reminders: store,
		APIToken:  "segredo",
	}).Router()
}

func reqWithAuth(method, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	authServer().ServeHTTP(rec, req)
	return rec
}

// TestAuth_DesligadaSemToken: APIToken vazio (dev local) → middleware é no-op.
func TestAuth_DesligadaSemToken(t *testing.T) {
	store := newFakeStore()
	h := (&Server{Boards: store, Columns: store, Cards: store, Reminders: store}).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/boards", nil) // sem Authorization
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth desligada: esperado 200 sem token, veio %d", rec.Code)
	}
}

// TestAuth_TokenAusente_401: auth ligada e sem header → 401.
func TestAuth_TokenAusente_401(t *testing.T) {
	rec := reqWithAuth(http.MethodGet, "/api/boards", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401 sem token, veio %d", rec.Code)
	}
}

// TestAuth_SemPrefixoBearer_401: header sem "Bearer " → 401.
func TestAuth_SemPrefixoBearer_401(t *testing.T) {
	rec := reqWithAuth(http.MethodGet, "/api/boards", "segredo")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401 sem prefixo Bearer, veio %d", rec.Code)
	}
}

// TestAuth_TokenInvalido_401: Bearer com token errado → 401.
func TestAuth_TokenInvalido_401(t *testing.T) {
	rec := reqWithAuth(http.MethodGet, "/api/boards", "Bearer errado")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401 com token inválido, veio %d", rec.Code)
	}
}

// TestAuth_TokenValido_OK: Bearer com o token certo → passa.
func TestAuth_TokenValido_OK(t *testing.T) {
	rec := reqWithAuth(http.MethodGet, "/api/boards", "Bearer segredo")
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200 com token válido, veio %d (body=%s)", rec.Code, rec.Body.String())
	}
}

// TestAuth_HealthPublico: /api/health é público mesmo com auth ligada (liveness).
func TestAuth_HealthPublico(t *testing.T) {
	rec := reqWithAuth(http.MethodGet, "/api/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("health deveria ser público, veio %d", rec.Code)
	}
}
