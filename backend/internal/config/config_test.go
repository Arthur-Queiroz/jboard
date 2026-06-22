package config

import (
	"strings"
	"testing"
)

// TestLoad_SenhaObrigatoria: sem JBOARD_DB_PASSWORD, Load falha.
func TestLoad_SenhaObrigatoria(t *testing.T) {
	t.Setenv("JBOARD_DB_PASSWORD", "") // set vazio = ausente para o Load
	if _, err := Load(); err == nil {
		t.Fatal("esperado erro com JBOARD_DB_PASSWORD vazio")
	}
}

// TestLoad_Defaults: com a senha definida, os defaults de dev são aplicados.
func TestLoad_Defaults(t *testing.T) {
	t.Setenv("JBOARD_DB_PASSWORD", "secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DBHost != "localhost" || cfg.DBPort != 5432 || cfg.ServerPort != 8080 {
		t.Fatalf("defaults inesperados: %+v", cfg)
	}
	// Default de CORS cobre as origens do desktop Tauri.
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "tauri://localhost" {
		t.Fatalf("AllowedOrigins default inesperado: %v", cfg.AllowedOrigins)
	}
}

// TestLoad_CORSCustom: lista separada por vírgula, ignorando espaços e vazios.
func TestLoad_CORSCustom(t *testing.T) {
	t.Setenv("JBOARD_DB_PASSWORD", "secret")
	t.Setenv("JBOARD_CORS_ORIGINS", " https://a.com , ,https://b.com ")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := strings.Join(cfg.AllowedOrigins, "|")
	if got != "https://a.com|https://b.com" {
		t.Fatalf("AllowedOrigins parseado errado: %q", got)
	}
}

// TestDSN: a string de conexão tem o formato esperado pelo driver pgx.
func TestDSN(t *testing.T) {
	t.Setenv("JBOARD_DB_PASSWORD", "secret")
	cfg, _ := Load()
	dsn := cfg.DSN()
	for _, want := range []string{"host=localhost", "port=5432", "user=jboard", "password=secret", "dbname=jboard", "sslmode=disable"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("DSN não contém %q: %s", want, dsn)
		}
	}
}
