package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
)

// newTestServer monta um Server apontando pro fakeStore, retornando o handler
// pronto pra httptest.
func newTestServer(t *testing.T) (http.Handler, *fakeStore) {
	t.Helper()
	store := newFakeStore()
	srv := &Server{
		Boards:    store,
		Columns:   store,
		Cards:     store,
		Reminders: store,
	}
	return srv.Router(), store
}

func doRequest(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var got T
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rec.Body.String())
	}
	return got
}

func TestCreateBoard_Validation(t *testing.T) {
	h, _ := newTestServer(t)

	rec := doRequest(t, h, http.MethodPost, "/api/boards", map[string]string{"title": "  "})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400 pra title vazio, veio %d (body=%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "title") {
		t.Fatalf("mensagem de erro deveria citar 'title': %s", rec.Body.String())
	}
}

func TestCreateBoard_OK(t *testing.T) {
	h, _ := newTestServer(t)

	rec := doRequest(t, h, http.MethodPost, "/api/boards", map[string]string{"title": "Estudos"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, veio %d", rec.Code)
	}
	board := decodeBody[domain.Board](t, rec)
	if board.ID == 0 || board.Title != "Estudos" {
		t.Fatalf("board criado inconsistente: %+v", board)
	}
}

func TestGetBoard_NotFound(t *testing.T) {
	h, _ := newTestServer(t)

	rec := doRequest(t, h, http.MethodGet, "/api/boards/999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404 pra board inexistente, veio %d", rec.Code)
	}
}

// TestKanbanFlow cobre o caminho feliz do kanban: board → column → cards →
// reorder → GetBoard devolve tudo na ordem esperada.
func TestKanbanFlow_ReorderPersists(t *testing.T) {
	h, store := newTestServer(t)

	// Cria board e coluna direto no store pra focar o teste no fluxo HTTP.
	board := &domain.Board{Title: "Estudos"}
	if err := store.CreateBoard(context.Background(), board); err != nil {
		t.Fatal(err)
	}
	column := &domain.Column{BoardID: board.ID, Title: "A fazer", Position: 0}
	if err := store.CreateColumn(context.Background(), column); err != nil {
		t.Fatal(err)
	}

	// Cria 3 cards via API (ordem de criação: A, B, C → positions 0,1,2 esperadas).
	for _, title := range []string{"A", "B", "C"} {
		rec := doRequest(t, h, http.MethodPost, "/api/columns/"+utoa(column.ID)+"/cards",
			map[string]any{"title": title, "position": 0})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create card %s: %d (body=%s)", title, rec.Code, rec.Body.String())
		}
	}

	cards, _ := store.ListCards(context.Background(), column.ID)
	if len(cards) != 3 {
		t.Fatalf("esperado 3 cards, veio %d", len(cards))
	}
	// IDs criados na ordem A, B, C.
	idA, idB, idC := cards[0].ID, cards[1].ID, cards[2].ID

	// Reorder: inverte pra C, A, B.
	rec := doRequest(t, h, http.MethodPost, "/api/columns/"+utoa(column.ID)+"/cards/reorder",
		map[string]any{"card_ids": []uint{idC, idA, idB}})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reorder: esperado 204, veio %d (body=%s)", rec.Code, rec.Body.String())
	}

	// GetBoard devolve a coluna com cards na nova ordem (position ASC, id ASC).
	boardRec := doRequest(t, h, http.MethodGet, "/api/boards/"+utoa(board.ID), nil)
	gotBoard := decodeBody[domain.Board](t, boardRec)
	if len(gotBoard.Columns) != 1 || len(gotBoard.Columns[0].Cards) != 3 {
		t.Fatalf("estrutura inconsistente: %+v", gotBoard)
	}
	gotCards := gotBoard.Columns[0].Cards
	if gotCards[0].ID != idC || gotCards[1].ID != idA || gotCards[2].ID != idB {
		t.Fatalf("ordem após reorder incorreta: %+v", gotCards)
	}
}

func TestUpdateCard_PersistsDescription(t *testing.T) {
	h, store := newTestServer(t)

	board := &domain.Board{Title: "Estudos"}
	store.CreateBoard(context.Background(), board)
	column := &domain.Column{BoardID: board.ID, Title: "A fazer", Position: 0}
	store.CreateColumn(context.Background(), column)
	card := &domain.Card{ColumnID: column.ID, Title: "A", Position: 0}
	store.CreateCard(context.Background(), card)

	// PUT com description não-vazia e position=0 (zero value) — o bug do GORM
	// com struct seria pular position=0; o Store usa map, então deve persistir.
	rec := doRequest(t, h, http.MethodPut, "/api/cards/"+utoa(card.ID),
		map[string]any{"title": "A revisado", "description": "anotações", "position": 0})
	if rec.Code != http.StatusOK {
		t.Fatalf("update card: %d (body=%s)", rec.Code, rec.Body.String())
	}

	got, _ := store.GetCard(context.Background(), card.ID)
	if got.Description != "anotações" || got.Position != 0 || got.Title != "A revisado" {
		t.Fatalf("card não persistiu corretamente: %+v", got)
	}
}

func TestCreateCard_EmptyTitle_400(t *testing.T) {
	h, store := newTestServer(t)

	column := &domain.Column{BoardID: 1, Title: "A fazer", Position: 0}
	store.CreateColumn(context.Background(), column)

	rec := doRequest(t, h, http.MethodPost, "/api/columns/"+utoa(column.ID)+"/cards",
		map[string]any{"title": ""})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, veio %d (body=%s)", rec.Code, rec.Body.String())
	}
}

// utoa evita importar strconv só pra converter uint pra string no path.
func utoa(id uint) string {
	if id == 0 {
		return "0"
	}
	var digits []byte
	for id > 0 {
		digits = append([]byte{byte('0' + id%10)}, digits...)
		id /= 10
	}
	return string(digits)
}
