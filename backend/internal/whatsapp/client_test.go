package whatsapp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSend_MontaRequest: o client deve postar em
// {baseURL}/message/sendText/{instance} com header apikey e corpo {number,text}.
func TestSend_MontaRequest(t *testing.T) {
	var gotPath, gotKey, gotCT string
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("apikey")
		gotCT = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewEvolutionClient(srv.URL, "inspire", "k3y")
	if err := c.Send(context.Background(), "5511999999999", "olá"); err != nil {
		t.Fatalf("Send: erro inesperado: %v", err)
	}

	if gotPath != "/message/sendText/inspire" {
		t.Fatalf("path errado: %q", gotPath)
	}
	if gotKey != "k3y" {
		t.Fatalf("header apikey errado: %q", gotKey)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type errado: %q", gotCT)
	}
	if gotBody["number"] != "5511999999999" || gotBody["text"] != "olá" {
		t.Fatalf("corpo errado: %+v", gotBody)
	}
}

// TestSend_StatusNao2xx_Erro: resposta >= 300 vira erro (lembrete não conta como
// enviado, então o scheduler tenta de novo).
func TestSend_StatusNao2xx_Erro(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewEvolutionClient(srv.URL, "inspire", "k3y")
	if err := c.Send(context.Background(), "5511999999999", "x"); err == nil {
		t.Fatal("esperado erro para status 500, veio nil")
	}
}

// TestSend_SemApiKey_OmiteHeader: apiKey vazio → não envia o header apikey.
func TestSend_SemApiKey_OmiteHeader(t *testing.T) {
	hadKey := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadKey = r.Header["Apikey"]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewEvolutionClient(srv.URL, "inspire", "")
	if err := c.Send(context.Background(), "5511999999999", "x"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if hadKey {
		t.Fatal("apiKey vazio não deveria mandar header apikey")
	}
}
