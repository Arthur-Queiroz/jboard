package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/Arthur-Queiroz/jboard/internal/repository"
	"github.com/Arthur-Queiroz/jboard/internal/whatsapp"
)

// Scheduler varre lembretes pendentes num ticker de 1 minuto e dispara via
// WhatsApp. Roda dentro do próprio processo do backend — é a única fonte de
// verdade, já que o lembrete precisa disparar mesmo com o desktop fechado.
type Scheduler struct {
	reminders repository.ReminderRepository
	sender    whatsapp.Sender
	interval  time.Duration
}

func New(reminders repository.ReminderRepository, sender whatsapp.Sender) *Scheduler {
	return &Scheduler{
		reminders: reminders,
		sender:    sender,
		interval:  time.Minute,
	}
}

// Start lança o ticker em segundo plano até ctx ser cancelado.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	go func() {
		// Roda imediatamente na subida, sem esperar o primeiro tick.
		s.tick(ctx)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.tick(ctx)
			}
		}
	}()
}

func (s *Scheduler) tick(ctx context.Context) {
	pending, err := s.reminders.ListPending(ctx, time.Now())
	if err != nil {
		log.Printf("scheduler: listar pendentes: %v", err)
		return
	}

	for _, reminder := range pending {
		// Envia antes de marcar: se o MarkSent falhar (transient no DB), o pior
		// caso é uma mensagem duplicada no próximo tick — nunca um lembrete
		// perdido. O claim atômico do MarkSent (WHERE sent_at IS NULL) evita
		// duplicação entre processos caso uma segunda instância do backend exista.
		if err := s.sender.Send(ctx, reminder.Recipient, reminder.Message); err != nil {
			log.Printf("scheduler: enviar lembrete %d: %v", reminder.ID, err)
			continue
		}
		if err := s.reminders.MarkSent(ctx, reminder.ID, time.Now()); err != nil && err != repository.ErrAlreadySent {
			log.Printf("scheduler: marcar enviado %d: %v", reminder.ID, err)
		}
	}
}
