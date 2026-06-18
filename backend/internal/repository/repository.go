package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
	"gorm.io/gorm"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrAlreadySent = errors.New("reminder already sent")
)

// BoardRepository cobre o CRUD de boards.
type BoardRepository interface {
	CreateBoard(ctx context.Context, board *domain.Board) error
	GetBoard(ctx context.Context, id uint) (*domain.Board, error)
	ListBoards(ctx context.Context) ([]domain.Board, error)
	UpdateBoard(ctx context.Context, board *domain.Board) error
	DeleteBoard(ctx context.Context, id uint) error
}

// ColumnRepository cobre o CRUD de colunas.
type ColumnRepository interface {
	CreateColumn(ctx context.Context, column *domain.Column) error
	GetColumn(ctx context.Context, id uint) (*domain.Column, error)
	ListColumns(ctx context.Context, boardID uint) ([]domain.Column, error)
	UpdateColumn(ctx context.Context, column *domain.Column) error
	DeleteColumn(ctx context.Context, id uint) error
}

// CardRepository cobre o CRUD de cards.
type CardRepository interface {
	CreateCard(ctx context.Context, card *domain.Card) error
	GetCard(ctx context.Context, id uint) (*domain.Card, error)
	ListCards(ctx context.Context, columnID uint) ([]domain.Card, error)
	UpdateCard(ctx context.Context, card *domain.Card) error
	// ReorderCards fixa a ordem dos cards numa coluna: o card em cardIDs[i]
	// recebe position=i e column_id=columnID. Em transação, pra não deixar
	// posições inconsistentes no meio se algo falhar. Coberto pelo DnD: o
	// frontend chama reorder na coluna de origem e na de destino após um move.
	ReorderCards(ctx context.Context, columnID uint, cardIDs []uint) error
	DeleteCard(ctx context.Context, id uint) error
}

// ReminderRepository inclui os métodos que o scheduler precisa além do CRUD:
// ListPending (lembretes vencidos e não enviados) e MarkSent (claim atômica).
type ReminderRepository interface {
	CreateReminder(ctx context.Context, reminder *domain.Reminder) error
	GetReminder(ctx context.Context, id uint) (*domain.Reminder, error)
	ListReminders(ctx context.Context, cardID uint) ([]domain.Reminder, error)
	ListPending(ctx context.Context, before time.Time) ([]domain.Reminder, error)
	MarkSent(ctx context.Context, id uint, at time.Time) error
	UpdateReminder(ctx context.Context, reminder *domain.Reminder) error
	DeleteReminder(ctx context.Context, id uint) error
}

// Store é a implementação concreta das interfaces sobre GORM. Os handlers e o
// scheduler dependem das interfaces, não do Store, o que permite mocks em teste.
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// --- Board ---

func (s *Store) CreateBoard(ctx context.Context, board *domain.Board) error {
	return s.db.WithContext(ctx).Create(board).Error
}

func (s *Store) GetBoard(ctx context.Context, id uint) (*domain.Board, error) {
	var board domain.Board
	err := s.db.WithContext(ctx).
		Preload("Columns.Cards.Reminders").
		First(&board, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &board, nil
}

func (s *Store) ListBoards(ctx context.Context) ([]domain.Board, error) {
	var boards []domain.Board
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&boards).Error
	return boards, err
}

func (s *Store) UpdateBoard(ctx context.Context, board *domain.Board) error {
	res := s.db.WithContext(ctx).Model(&domain.Board{}).
		Where("id = ?", board.ID).Updates(board)
	return wrapUpdate(res)
}

func (s *Store) DeleteBoard(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&domain.Board{}, id)
	return wrapDelete(res)
}

// --- Column ---

func (s *Store) CreateColumn(ctx context.Context, column *domain.Column) error {
	return s.db.WithContext(ctx).Create(column).Error
}

func (s *Store) GetColumn(ctx context.Context, id uint) (*domain.Column, error) {
	var column domain.Column
	err := s.db.WithContext(ctx).First(&column, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &column, nil
}

func (s *Store) ListColumns(ctx context.Context, boardID uint) ([]domain.Column, error) {
	var columns []domain.Column
	err := s.db.WithContext(ctx).
		Where("board_id = ?", boardID).
		Order("position ASC, id ASC").Find(&columns).Error
	return columns, err
}

func (s *Store) UpdateColumn(ctx context.Context, column *domain.Column) error {
	// Map pra permitir position=0 (primeira coluna).
	updates := map[string]any{
		"title":    column.Title,
		"position": column.Position,
		"board_id": column.BoardID,
	}
	res := s.db.WithContext(ctx).Model(&domain.Column{}).
		Where("id = ?", column.ID).Updates(updates)
	return wrapUpdate(res)
}

func (s *Store) DeleteColumn(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&domain.Column{}, id)
	return wrapDelete(res)
}

// --- Card ---

func (s *Store) CreateCard(ctx context.Context, card *domain.Card) error {
	return s.db.WithContext(ctx).Create(card).Error
}

func (s *Store) GetCard(ctx context.Context, id uint) (*domain.Card, error) {
	var card domain.Card
	err := s.db.WithContext(ctx).First(&card, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &card, nil
}

func (s *Store) ListCards(ctx context.Context, columnID uint) ([]domain.Card, error) {
	var cards []domain.Card
	err := s.db.WithContext(ctx).
		Where("column_id = ?", columnID).
		Order("position ASC, id ASC").Find(&cards).Error
	return cards, err
}

func (s *Store) UpdateCard(ctx context.Context, card *domain.Card) error {
	// Map em vez de struct: GORM com struct pula campos zero (position=0,
	// description=""), o que quebra reorder pro topo e limpar descrição.
	updates := map[string]any{
		"title":       card.Title,
		"description": card.Description,
		"position":    card.Position,
		"column_id":   card.ColumnID,
	}
	res := s.db.WithContext(ctx).Model(&domain.Card{}).
		Where("id = ?", card.ID).Updates(updates)
	return wrapUpdate(res)
}

// ReorderCards fixa position=i e column_id=columnID pra cada cardIDs[i], em
// transação. Map pra persistir position=0 (topo da coluna).
func (s *Store) ReorderCards(ctx context.Context, columnID uint, cardIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for position, id := range cardIDs {
			updates := map[string]any{
				"column_id": columnID,
				"position":  position,
			}
			if err := tx.Model(&domain.Card{}).
				Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) DeleteCard(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&domain.Card{}, id)
	return wrapDelete(res)
}

// --- Reminder ---

func (s *Store) CreateReminder(ctx context.Context, reminder *domain.Reminder) error {
	return s.db.WithContext(ctx).Create(reminder).Error
}

func (s *Store) GetReminder(ctx context.Context, id uint) (*domain.Reminder, error) {
	var reminder domain.Reminder
	err := s.db.WithContext(ctx).First(&reminder, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &reminder, nil
}

func (s *Store) ListReminders(ctx context.Context, cardID uint) ([]domain.Reminder, error) {
	var reminders []domain.Reminder
	err := s.db.WithContext(ctx).
		Where("card_id = ?", cardID).
		Order("reminder_at ASC").Find(&reminders).Error
	return reminders, err
}

// ListPending retorna lembretes vencidos (reminder_at <= before) ainda não
// enviados (sent_at IS NULL).
func (s *Store) ListPending(ctx context.Context, before time.Time) ([]domain.Reminder, error) {
	var reminders []domain.Reminder
	err := s.db.WithContext(ctx).
		Where("sent_at IS NULL AND reminder_at <= ?", before).
		Order("reminder_at ASC").Find(&reminders).Error
	return reminders, err
}

// MarkSent faz o claim atômico: só atualiza se sent_at continua NULL. Se outro
// processo (ou um tick concorrente) já marcou, RowsAffected = 0 -> ErrAlreadySent.
func (s *Store) MarkSent(ctx context.Context, id uint, at time.Time) error {
	res := s.db.WithContext(ctx).Model(&domain.Reminder{}).
		Where("id = ? AND sent_at IS NULL", id).
		Update("sent_at", at)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAlreadySent
	}
	return nil
}

func (s *Store) DeleteReminder(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&domain.Reminder{}, id)
	return wrapDelete(res)
}

func (s *Store) UpdateReminder(ctx context.Context, reminder *domain.Reminder) error {
	res := s.db.WithContext(ctx).Model(&domain.Reminder{}).
		Where("id = ?", reminder.ID).Updates(reminder)
	return wrapUpdate(res)
}

// wrapUpdate traduz "0 linhas afetadas" como ErrNotFound, pra o handler devolver
// 404 em PUT num recurso inexistente.
func wrapUpdate(res *gorm.DB) error {
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func wrapDelete(res *gorm.DB) error {
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
