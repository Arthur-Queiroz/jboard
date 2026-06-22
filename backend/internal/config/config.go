package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config reúne tudo que o backend precisa para subir. Tudo vem do ambiente,
// com defaults voltados pro desenvolvimento local contra o docker-compose.
type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	ServerPort int

	EvolutionBaseURL  string
	EvolutionInstance string
	EvolutionAPIKey   string

	// WhatsAppRecipient é o destinatário padrão dos lembretes (número pessoal),
	// usado quando o card não define um recipient explícito.
	WhatsAppRecipient string

	// APIToken protege a API REST em deploy público. Vazio = auth desligada
	// (dev local). Em produção, o frontend envia o token no header
	// Authorization: Bearer <token>.
	APIToken string

	// AllowedOrigins é a allowlist de CORS. A web e o desktop em dev falam com a
	// API na mesma origem (proxy do Vite / Caddy), então não precisam de CORS; só
	// o desktop empacotado, cujo webview tem origem própria (tauri://localhost no
	// Linux/macOS, http://tauri.localhost no Windows). Default cobre esses casos.
	AllowedOrigins []string
}

func Load() (Config, error) {
	cfg := Config{
		DBHost:            envStr("JBOARD_DB_HOST", "localhost"),
		DBPort:            envInt("JBOARD_DB_PORT", 5432),
		DBUser:            envStr("JBOARD_DB_USER", "jboard"),
		DBPassword:        envStr("JBOARD_DB_PASSWORD", ""),
		DBName:            envStr("JBOARD_DB_NAME", "jboard"),
		DBSSLMode:         envStr("JBOARD_DB_SSLMODE", "disable"),
		ServerPort:        envInt("JBOARD_SERVER_PORT", 8080),
		EvolutionBaseURL:  envStr("JBOARD_EVOLUTION_URL", "http://evolution-api:8080"),
		EvolutionInstance: envStr("JBOARD_EVOLUTION_INSTANCE", ""),
		EvolutionAPIKey:   envStr("JBOARD_EVOLUTION_API_KEY", ""),
		WhatsAppRecipient: envStr("JBOARD_WHATSAPP_RECIPIENT", ""),
		APIToken:          envStr("JBOARD_API_TOKEN", ""),
		AllowedOrigins:    envStrSlice("JBOARD_CORS_ORIGINS", "tauri://localhost,http://tauri.localhost"),
	}
	if cfg.DBPassword == "" {
		return cfg, fmt.Errorf("JBOARD_DB_PASSWORD é obrigatório")
	}
	return cfg, nil
}

// DSN monta a string de conexão no formato esperado pelo driver postgres do GORM
// (pgx pure-Go, sem CGO).
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func envStr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// envStrSlice lê uma lista separada por vírgula, ignorando espaços e itens vazios.
func envStrSlice(key, fallback string) []string {
	raw := envStr(key, fallback)
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fallback
		}
		return n
	}
	return fallback
}
