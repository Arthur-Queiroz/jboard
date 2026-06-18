package db

import (
	"embed"
	"fmt"

	"github.com/Arthur-Queiroz/jboard/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// migrationsFS embarca os arquivos .up.sql/.down.sql no binário, dispensando
// CLI externa pra aplicar migrations no boot do servidor.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect abre a conexão com o Postgres via GORM, aplica as migrations
// versionadas (golang-migrate) e devolve o *gorm.DB pra uso no app.
//
// As migrations vivem em migrations/*.sql e são embarcadas no binário. O
// AutoMigrate do GORM foi substituído pra evitar drift entre desenvolvimento
// e produção — qualquer alteração de schema passa por uma nova migration.
func Connect(cfg config.Config) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	// golang-migrate precisa de URL no formato postgres://... (não key=value).
	migrateURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
	)
	if err := runMigrations(migrateURL); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return gormDB, nil
}

// ConnectWithDSN é a entrada de baixo nível: aceita a DSN pronta no formato
// postgres://... (usada por testes de integração com testcontainers). Aplica
// as migrations e devolve o *gorm.DB.
func ConnectWithDSN(dsn string) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return gormDB, nil
}

// runMigrations aplica as migrations embarcadas via iofs. Usa
// NewWithSourceInstance (source Driver já construído + database URL string,
// que o adapter postgres do migrate parseia sozinho). Idempotente: ErrNoChange
// significa que o banco já está na versão mais recente.
//
// A DSN deve estar no formato postgres://user:pass@host:port/db?sslmode=...
func runMigrations(dsn string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
