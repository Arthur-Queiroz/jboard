package db

import (
	"context"
	"testing"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
	"github.com/Arthur-Queiroz/jboard/internal/repository"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestStore_Integration sobe um Postgres real via testcontainers, aplica as
// migrations, e exercita o Store (CreateBoard, GetBoard, ReorderCards) pra
// validar que o SQL e o GORM conversam corretamente com o banco de verdade.
//
// Pula automaticamente se Docker não estiver disponível (CI sem Docker).
func TestStore_Integration(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := pgcontainer.Run(ctx,
		"postgres:16",
		pgcontainer.WithDatabase("jboard_test"),
		pgcontainer.WithUsername("jboard"),
		pgcontainer.WithPassword("jboard"),
		pgcontainer.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("postgres container indisponível (Docker offline?): %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	// ConnectWithDSN aplica as migrations e devolve o *gorm.DB, usando a
	// connStr do container diretamente (sem passar por config.Config.DSN()).
	gormDB, err := ConnectWithDSN(connStr)
	if err != nil {
		t.Fatalf("connect+migrate: %v", err)
	}

	store := repository.NewStore(gormDB)

	// Cria board → column → 3 cards.
	board := &domain.Board{Title: "Testes"}
	if err := store.CreateBoard(ctx, board); err != nil {
		t.Fatalf("create board: %v", err)
	}
	column := &domain.Column{BoardID: board.ID, Title: "A fazer", Position: 0}
	if err := store.CreateColumn(ctx, column); err != nil {
		t.Fatalf("create column: %v", err)
	}

	for _, title := range []string{"A", "B", "C"} {
		card := &domain.Card{ColumnID: column.ID, Title: title, Position: 0}
		if err := store.CreateCard(ctx, card); err != nil {
			t.Fatalf("create card %s: %v", title, err)
		}
	}

	cards, err := store.ListCards(ctx, column.ID)
	if err != nil {
		t.Fatalf("list cards: %v", err)
	}
	if len(cards) != 3 {
		t.Fatalf("esperado 3 cards, veio %d", len(cards))
	}

	// Reorder: inverte a ordem e valida que persistiu.
	newOrder := []uint{cards[2].ID, cards[0].ID, cards[1].ID}
	if err := store.ReorderCards(ctx, column.ID, newOrder); err != nil {
		t.Fatalf("reorder: %v", err)
	}

	reordered, _ := store.ListCards(ctx, column.ID)
	for i, expected := range newOrder {
		if reordered[i].ID != expected {
			t.Fatalf("posição %d: esperado card %d, veio %d", i, expected, reordered[i].ID)
		}
	}

	// GetBoard traz o board com columns.cards aninhados (Preload).
	gotBoard, err := store.GetBoard(ctx, board.ID)
	if err != nil {
		t.Fatalf("get board: %v", err)
	}
	if len(gotBoard.Columns) != 1 || len(gotBoard.Columns[0].Cards) != 3 {
		t.Fatalf("preload inconsistente: %+v", gotBoard)
	}

	// DeleteBoard cascata.
	if err := store.DeleteBoard(ctx, board.ID); err != nil {
		t.Fatalf("delete board: %v", err)
	}
	if _, err := store.GetBoard(ctx, board.ID); err != repository.ErrNotFound {
		t.Fatalf("esperado ErrNotFound após delete, veio %v", err)
	}
}
