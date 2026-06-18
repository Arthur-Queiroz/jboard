package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Arthur-Queiroz/jboard/internal/api"
	"github.com/Arthur-Queiroz/jboard/internal/config"
	"github.com/Arthur-Queiroz/jboard/internal/db"
	"github.com/Arthur-Queiroz/jboard/internal/repository"
	"github.com/Arthur-Queiroz/jboard/internal/scheduler"
	"github.com/Arthur-Queiroz/jboard/internal/whatsapp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	gormDB, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	store := repository.NewStore(gormDB)

	sender := whatsapp.NewEvolutionClient(cfg.EvolutionBaseURL, cfg.EvolutionInstance, cfg.EvolutionAPIKey)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	scheduler.New(store, sender).Start(ctx)

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.ServerPort),
		Handler: (&api.Server{
			Boards:           store,
			Columns:          store,
			Cards:            store,
			Reminders:        store,
			DefaultRecipient: cfg.WhatsAppRecipient,
			APIToken:         cfg.APIToken,
		}).Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("jboard backend escutando em %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("encerrando...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
