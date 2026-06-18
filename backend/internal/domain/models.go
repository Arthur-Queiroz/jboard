package domain

import "time"

// Board é o quadro kanban. Agrupa Column -> Card -> Reminder.
type Board struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"not null" json:"title"`
	Columns   []Column  `gorm:"foreignKey:BoardID;constraint:OnDelete:CASCADE" json:"columns"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Column é uma coluna do quadro (ex: "A fazer", "Fazendo", "Feito").
type Column struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BoardID   uint      `gorm:"not null" json:"board_id"`
	Title     string    `gorm:"not null" json:"title"`
	Position  int       `json:"position"`
	Cards     []Card    `gorm:"foreignKey:ColumnID;constraint:OnDelete:CASCADE" json:"cards"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Card é um item dentro de uma coluna.
type Card struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ColumnID    uint       `gorm:"not null" json:"column_id"`
	Title       string     `gorm:"not null" json:"title"`
	Description string     `json:"description"`
	Position    int        `json:"position"`
	Reminders   []Reminder `gorm:"foreignKey:CardID;constraint:OnDelete:CASCADE" json:"reminders"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Reminder é o agendamento de um lembrete via WhatsApp para um card.
//
// SentAt é nil enquanto o lembrete está pendente; o scheduler o preenche no
// momento do envio. A coluna é a base da idempotência (ver scheduler).
type Reminder struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	CardID     uint       `gorm:"not null" json:"card_id"`
	ReminderAt time.Time  `gorm:"not null" json:"reminder_at"`
	Recipient  string     `json:"recipient"`
	Message    string     `json:"message"`
	SentAt     *time.Time `json:"sent_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
