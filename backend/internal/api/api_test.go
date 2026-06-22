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
	"time"

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

// TestCreateCard_RemindersIsArray regression: o handler precisa devolver
// reminders como [] (não null). nil slice em Go vira "null" no JSON, o que
// quebra o frontend (card.reminders.length lança TypeError).
func TestCreateCard_RemindersIsArray(t *testing.T) {
	h, store := newTestServer(t)

	column := &domain.Column{BoardID: 1, Title: "A fazer", Position: 0}
	store.CreateColumn(context.Background(), column)

	rec := doRequest(t, h, http.MethodPost, "/api/columns/"+utoa(column.ID)+"/cards",
		map[string]any{"title": "X", "position": 0})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create card: %d (body=%s)", rec.Code, rec.Body.String())
	}
	card := decodeBody[domain.Card](t, rec)
	if card.Reminders == nil {
		t.Fatalf("reminders devolvido como null; esperado [] — body=%s", rec.Body.String())
	}
}

// TestCORS_PreflightAllowedOrigin: o webview do desktop manda um preflight OPTIONS
// com Origin tauri://localhost; o backend deve responder 204 com os headers de CORS
// ecoando a origem, sem passar pela auth.
func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	srv := &Server{
		Boards:         newFakeStore(),
		Columns:        nil,
		Cards:          nil,
		Reminders:      nil,
		APIToken:       "segredo", // auth ligada: o preflight não pode esbarrar nela
		AllowedOrigins: []string{"tauri://localhost"},
	}
	h := srv.Router()

	req := httptest.NewRequest(http.MethodOptions, "/api/boards", nil)
	req.Header.Set("Origin", "tauri://localhost")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight: esperado 204, veio %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "tauri://localhost" {
		t.Fatalf("Allow-Origin: esperado tauri://localhost, veio %q", got)
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("Allow-Headers deveria incluir Authorization, veio %q", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

// TestCORS_DisallowedOrigin: origem fora da allowlist não recebe header de CORS.
func TestCORS_DisallowedOrigin(t *testing.T) {
	srv := &Server{Boards: newFakeStore(), AllowedOrigins: []string{"tauri://localhost"}}
	h := srv.Router()

	req := httptest.NewRequest(http.MethodOptions, "/api/boards", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("origem não-permitida não deveria receber Allow-Origin, veio %q", got)
	}
}

// TestMaxBodyBytes_CorpoGigante400: body acima do limite faz o decode falhar → 400.
func TestMaxBodyBytes_CorpoGigante400(t *testing.T) {
	h, _ := newTestServer(t)

	body := `{"title":"` + strings.Repeat("a", 2<<20) + `"}` // ~2 MB > limite de 1 MB
	req := httptest.NewRequest(http.MethodPost, "/api/boards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("corpo gigante: esperado 400, veio %d", rec.Code)
	}
}

// TestColumn_UpdateDelete: PUT (200, persiste título) e DELETE (204) de coluna.
func TestColumn_UpdateDelete(t *testing.T) {
	h, store := newTestServer(t)
	col := &domain.Column{BoardID: 1, Title: "A", Position: 0}
	store.CreateColumn(context.Background(), col)

	rec := doRequest(t, h, http.MethodPut, "/api/columns/"+utoa(col.ID), map[string]any{"title": "B", "position": 1})
	if rec.Code != http.StatusOK {
		t.Fatalf("update column: %d", rec.Code)
	}
	if got := decodeBody[domain.Column](t, rec); got.Title != "B" {
		t.Fatalf("título não atualizou: %+v", got)
	}

	rec = doRequest(t, h, http.MethodDelete, "/api/columns/"+utoa(col.ID), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete column: %d", rec.Code)
	}
}

// TestColumn_Update_NotFound: PUT em coluna inexistente → 404.
func TestColumn_Update_NotFound(t *testing.T) {
	h, _ := newTestServer(t)
	rec := doRequest(t, h, http.MethodPut, "/api/columns/999", map[string]any{"title": "X"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, veio %d", rec.Code)
	}
}

// TestReminder_CreateUsaDefaultRecipient: sem recipient no corpo, usa o padrão da
// config; reminder novo nasce sem sent_at. Cobre também o DELETE.
func TestReminder_CreateUsaDefaultRecipient(t *testing.T) {
	store := newFakeStore()
	h := (&Server{Boards: store, Columns: store, Cards: store, Reminders: store, DefaultRecipient: "5511888888888"}).Router()
	card := &domain.Card{ColumnID: 1, Title: "c"}
	store.CreateCard(context.Background(), card)

	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	rec := doRequest(t, h, http.MethodPost, "/api/cards/"+utoa(card.ID)+"/reminders",
		map[string]any{"reminder_at": future, "message": "oi"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create reminder: %d (body=%s)", rec.Code, rec.Body.String())
	}
	got := decodeBody[domain.Reminder](t, rec)
	if got.Recipient != "5511888888888" {
		t.Fatalf("esperado default recipient, veio %q", got.Recipient)
	}
	if got.SentAt != nil {
		t.Fatal("reminder novo não deveria ter sent_at")
	}

	rec = doRequest(t, h, http.MethodDelete, "/api/reminders/"+utoa(got.ID), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete reminder: %d", rec.Code)
	}
}

// TestReminder_Create_PassadoRejeitado: reminder_at no passado → 400.
func TestReminder_Create_PassadoRejeitado(t *testing.T) {
	h, store := newTestServer(t)
	card := &domain.Card{ColumnID: 1, Title: "c"}
	store.CreateCard(context.Background(), card)

	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	rec := doRequest(t, h, http.MethodPost, "/api/cards/"+utoa(card.ID)+"/reminders",
		map[string]any{"reminder_at": past, "message": "oi"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400 para reminder no passado, veio %d", rec.Code)
	}
}

// TestBoard_UpdateDelete: update (200) e delete (204 + 404 depois) de board.
func TestBoard_UpdateDelete(t *testing.T) {
	h, store := newTestServer(t)
	b := &domain.Board{Title: "A"}
	store.CreateBoard(context.Background(), b)

	rec := doRequest(t, h, http.MethodPut, "/api/boards/"+utoa(b.ID), map[string]any{"title": "B"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update board: %d", rec.Code)
	}
	rec = doRequest(t, h, http.MethodDelete, "/api/boards/"+utoa(b.ID), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete board: %d", rec.Code)
	}
	rec = doRequest(t, h, http.MethodGet, "/api/boards/"+utoa(b.ID), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("board deletado deveria dar 404, veio %d", rec.Code)
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
