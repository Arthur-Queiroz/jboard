package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
	"github.com/Arthur-Queiroz/jboard/internal/repository"
)

// fakeReminders implementa repository.ReminderRepository in-memory. Só
// ListPending/MarkSent têm lógica real — é o que o scheduler usa. ListPending
// devolve os não enviados e vencidos; MarkSent faz o claim atômico (ErrAlreadySent
// se já enviado), espelhando o UPDATE ... WHERE sent_at IS NULL do Store.
type fakeReminders struct {
	items     map[uint]*domain.Reminder
	listErr   error
	markErr   error
	markCalls int
}

func newFakeReminders(rs ...*domain.Reminder) *fakeReminders {
	f := &fakeReminders{items: make(map[uint]*domain.Reminder)}
	for _, r := range rs {
		f.items[r.ID] = r
	}
	return f
}

func (f *fakeReminders) ListPending(_ context.Context, before time.Time) ([]domain.Reminder, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []domain.Reminder
	for _, r := range f.items {
		if r.SentAt == nil && r.ReminderAt.Before(before) {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (f *fakeReminders) MarkSent(_ context.Context, id uint, at time.Time) error {
	f.markCalls++
	if f.markErr != nil {
		return f.markErr
	}
	r, ok := f.items[id]
	if !ok || r.SentAt != nil {
		return repository.ErrAlreadySent // claim já feito por outro tick/instância
	}
	sent := at
	r.SentAt = &sent
	return nil
}

// resto da interface: não usado pelo scheduler.
func (f *fakeReminders) CreateReminder(context.Context, *domain.Reminder) error { return nil }
func (f *fakeReminders) GetReminder(context.Context, uint) (*domain.Reminder, error) {
	return nil, nil
}
func (f *fakeReminders) ListReminders(context.Context, uint) ([]domain.Reminder, error) {
	return nil, nil
}
func (f *fakeReminders) UpdateReminder(context.Context, *domain.Reminder) error { return nil }
func (f *fakeReminders) DeleteReminder(context.Context, uint) error             { return nil }

// fakeSender registra cada envio e pode simular falha.
type fakeSender struct {
	sent []string // recipients enviados, na ordem
	err  error
}

func (f *fakeSender) Send(_ context.Context, recipient, _ string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, recipient)
	return nil
}

func pastReminder(id uint) *domain.Reminder {
	return &domain.Reminder{
		ID:         id,
		Recipient:  "5511999999999",
		Message:    "lembrete",
		ReminderAt: time.Now().Add(-time.Minute), // vencido
	}
}

// TestTick_EnviaEMarcaUmaVez: lembrete vencido é enviado e marcado; um segundo
// tick NÃO reenvia (idempotência — a propriedade central do scheduler).
func TestTick_EnviaEMarcaUmaVez(t *testing.T) {
	repo := newFakeReminders(pastReminder(1))
	sender := &fakeSender{}
	s := New(repo, sender)

	s.tick(context.Background())
	if len(sender.sent) != 1 {
		t.Fatalf("1º tick: esperado 1 envio, veio %d", len(sender.sent))
	}
	if repo.items[1].SentAt == nil {
		t.Fatal("1º tick: reminder deveria estar marcado (SentAt != nil)")
	}

	s.tick(context.Background())
	if len(sender.sent) != 1 {
		t.Fatalf("2º tick: não deveria reenviar, total de envios=%d", len(sender.sent))
	}
}

// TestTick_FalhaNoEnvioNaoMarca: se o envio falha, o lembrete NÃO é marcado e
// continua pendente — garante que nunca se perde um lembrete por erro transitório.
func TestTick_FalhaNoEnvioNaoMarca(t *testing.T) {
	repo := newFakeReminders(pastReminder(1))
	sender := &fakeSender{err: errors.New("evolution fora do ar")}
	s := New(repo, sender)

	s.tick(context.Background())
	if repo.markCalls != 0 {
		t.Fatalf("envio falhou: MarkSent não deveria ser chamado, calls=%d", repo.markCalls)
	}
	if repo.items[1].SentAt != nil {
		t.Fatal("envio falhou: reminder não deveria estar marcado")
	}

	// Evolution volta: o próximo tick envia e marca.
	sender.err = nil
	s.tick(context.Background())
	if len(sender.sent) != 1 || repo.items[1].SentAt == nil {
		t.Fatalf("após recuperar: esperado envio+marca, sent=%d marcado=%v", len(sender.sent), repo.items[1].SentAt != nil)
	}
}

// TestTick_MarkSentAlreadySent_Ignorado: se outro tick/instância já marcou
// (MarkSent devolve ErrAlreadySent), o scheduler ignora silenciosamente — sem
// erro, sem reenvio em loop.
func TestTick_MarkSentAlreadySent_Ignorado(t *testing.T) {
	repo := newFakeReminders(pastReminder(1))
	repo.markErr = repository.ErrAlreadySent
	sender := &fakeSender{}
	s := New(repo, sender)

	s.tick(context.Background()) // não deve entrar em pânico nem propagar erro
	if len(sender.sent) != 1 {
		t.Fatalf("esperado 1 tentativa de envio, veio %d", len(sender.sent))
	}
	if repo.markCalls != 1 {
		t.Fatalf("esperado 1 chamada a MarkSent, veio %d", repo.markCalls)
	}
}

// TestTick_ListPendingErro_NaoEnvia: se a listagem falha, o tick aborta sem enviar.
func TestTick_ListPendingErro_NaoEnvia(t *testing.T) {
	repo := newFakeReminders(pastReminder(1))
	repo.listErr = errors.New("db indisponível")
	sender := &fakeSender{}
	s := New(repo, sender)

	s.tick(context.Background())
	if len(sender.sent) != 0 {
		t.Fatalf("erro ao listar: não deveria enviar nada, veio %d", len(sender.sent))
	}
}
