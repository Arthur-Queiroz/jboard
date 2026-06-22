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

// authServerComSenha monta um Server com Bearer e login por senha ligados.
func authServerComSenha() http.Handler {
	store := newFakeStore()
	return (&Server{
		Boards: store, Columns: store, Cards: store, Reminders: store,
		APIToken: "segredo", AuthPassword: "minhasenha",
	}).Router()
}

// TestLogin_OK_DefineCookie_EConcedeAcesso: senha certa devolve 200 + cookie de
// sessão; uma request com esse cookie (sem Bearer) é autorizada.
func TestLogin_OK_DefineCookie_EConcedeAcesso(t *testing.T) {
	h := authServerComSenha()

	rec := doRequest(t, h, http.MethodPost, "/api/login", map[string]string{"password": "minhasenha"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: esperado 200, veio %d", rec.Code)
	}
	cookie := rec.Result().Cookies()
	var session *http.Cookie
	for _, c := range cookie {
		if c.Name == sessionCookieName {
			session = c
		}
	}
	if session == nil || session.Value == "" {
		t.Fatal("login deveria setar o cookie de sessão")
	}
	if !session.HttpOnly {
		t.Fatal("cookie de sessão deveria ser HttpOnly")
	}

	// Agora acessa um recurso protegido só com o cookie (sem Bearer).
	req := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	req.AddCookie(session)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("cookie de sessão deveria autorizar, veio %d", rec2.Code)
	}
}

// TestLogin_SenhaErrada_401.
func TestLogin_SenhaErrada_401(t *testing.T) {
	h := authServerComSenha()
	rec := doRequest(t, h, http.MethodPost, "/api/login", map[string]string{"password": "errada"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401 com senha errada, veio %d", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("senha errada não deveria setar cookie")
	}
}

// TestLogin_SemSenhaConfigurada_401: AuthPassword vazio → login indisponível.
func TestLogin_SemSenhaConfigurada_401(t *testing.T) {
	rec := doRequest(t, authServer(), http.MethodPost, "/api/login", map[string]string{"password": "qualquer"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401 sem senha configurada, veio %d", rec.Code)
	}
}

// TestCookieInvalido_401: cookie de sessão forjado não autoriza.
func TestCookieInvalido_401(t *testing.T) {
	h := authServerComSenha()
	req := httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "forjado.invalido"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie forjado deveria dar 401, veio %d", rec.Code)
	}
}
