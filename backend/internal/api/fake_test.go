package api

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
	"github.com/Arthur-Queiroz/jboard/internal/repository"
)

// fakeStore é um repositório in-memory que implementa as 4 interfaces, pra
// testar os handlers sem Postgres. Mantém slices ordenados por position pra
// espelhar o comportamento do Store (ORDER BY position ASC, id ASC).
type fakeStore struct {
	mu        sync.Mutex
	boards    map[uint]*domain.Board
	columns   map[uint]*domain.Column
	cards     map[uint]*domain.Card
	reminders map[uint]*domain.Reminder
	nextID    uint
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		boards:    make(map[uint]*domain.Board),
		columns:   make(map[uint]*domain.Column),
		cards:     make(map[uint]*domain.Card),
		reminders: make(map[uint]*domain.Reminder),
		nextID:    1,
	}
}

func (f *fakeStore) id() uint {
	id := f.nextID
	f.nextID++
	return id
}

// --- Board ---

func (f *fakeStore) CreateBoard(_ context.Context, board *domain.Board) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	board.ID = f.id()
	now := time.Now().UTC()
	board.CreatedAt = now
	board.UpdatedAt = now
	f.boards[board.ID] = board
	return nil
}

func (f *fakeStore) GetBoard(_ context.Context, id uint) (*domain.Board, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	board, ok := f.boards[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	// Devolve cópia com columns.cards.reminders montados, como o Store faz com Preload.
	out := *board
	out.Columns = f.columnsOf(id)
	return &out, nil
}

func (f *fakeStore) ListBoards(_ context.Context) ([]domain.Board, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var boards []domain.Board
	for _, board := range f.boards {
		boards = append(boards, *board)
	}
	return boards, nil
}

func (f *fakeStore) UpdateBoard(_ context.Context, board *domain.Board) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.boards[board.ID]; !ok {
		return repository.ErrNotFound
	}
	board.UpdatedAt = time.Now().UTC()
	f.boards[board.ID] = board
	return nil
}

func (f *fakeStore) DeleteBoard(_ context.Context, id uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.boards[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.boards, id)
	// Cascata: remove columns (e seus cards/reminders) do board.
	for cid, column := range f.columns {
		if column.BoardID == id {
			f.deleteColumnCards(cid)
			delete(f.columns, cid)
		}
	}
	return nil
}

// --- Column ---

func (f *fakeStore) CreateColumn(_ context.Context, column *domain.Column) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	column.ID = f.id()
	now := time.Now().UTC()
	column.CreatedAt = now
	column.UpdatedAt = now
	f.columns[column.ID] = column
	return nil
}

func (f *fakeStore) GetColumn(_ context.Context, id uint) (*domain.Column, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	column, ok := f.columns[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return column, nil
}

func (f *fakeStore) ListColumns(_ context.Context, boardID uint) ([]domain.Column, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var columns []domain.Column
	for _, column := range f.columns {
		if column.BoardID == boardID {
			columns = append(columns, *column)
		}
	}
	sortColumns(columns)
	return columns, nil
}

func (f *fakeStore) UpdateColumn(_ context.Context, column *domain.Column) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.columns[column.ID]; !ok {
		return repository.ErrNotFound
	}
	column.UpdatedAt = time.Now().UTC()
	f.columns[column.ID] = column
	return nil
}

func (f *fakeStore) DeleteColumn(_ context.Context, id uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.columns[id]; !ok {
		return repository.ErrNotFound
	}
	f.deleteColumnCards(id)
	delete(f.columns, id)
	return nil
}

// --- Card ---

func (f *fakeStore) CreateCard(_ context.Context, card *domain.Card) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	card.ID = f.id()
	now := time.Now().UTC()
	card.CreatedAt = now
	card.UpdatedAt = now
	f.cards[card.ID] = card
	return nil
}

func (f *fakeStore) GetCard(_ context.Context, id uint) (*domain.Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	card, ok := f.cards[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return card, nil
}

func (f *fakeStore) ListCards(_ context.Context, columnID uint) ([]domain.Card, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var cards []domain.Card
	for _, card := range f.cards {
		if card.ColumnID == columnID {
			cards = append(cards, *card)
		}
	}
	sortCards(cards)
	return cards, nil
}

func (f *fakeStore) UpdateCard(_ context.Context, card *domain.Card) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.cards[card.ID]; !ok {
		return repository.ErrNotFound
	}
	card.UpdatedAt = time.Now().UTC()
	f.cards[card.ID] = card
	return nil
}

func (f *fakeStore) ReorderCards(_ context.Context, columnID uint, cardIDs []uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for position, id := range cardIDs {
		card, ok := f.cards[id]
		if !ok {
			return repository.ErrNotFound
		}
		card.ColumnID = columnID
		card.Position = position
	}
	return nil
}

func (f *fakeStore) DeleteCard(_ context.Context, id uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.cards[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.cards, id)
	return nil
}

// --- Reminder ---

func (f *fakeStore) CreateReminder(_ context.Context, reminder *domain.Reminder) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	reminder.ID = f.id()
	now := time.Now().UTC()
	reminder.CreatedAt = now
	reminder.UpdatedAt = now
	f.reminders[reminder.ID] = reminder
	return nil
}

func (f *fakeStore) GetReminder(_ context.Context, id uint) (*domain.Reminder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	reminder, ok := f.reminders[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return reminder, nil
}

func (f *fakeStore) ListReminders(_ context.Context, cardID uint) ([]domain.Reminder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var reminders []domain.Reminder
	for _, reminder := range f.reminders {
		if reminder.CardID == cardID {
			reminders = append(reminders, *reminder)
		}
	}
	return reminders, nil
}

func (f *fakeStore) ListPending(_ context.Context, before time.Time) ([]domain.Reminder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var pending []domain.Reminder
	for _, reminder := range f.reminders {
		if reminder.SentAt == nil && reminder.ReminderAt.Before(before) {
			pending = append(pending, *reminder)
		}
	}
	return pending, nil
}

func (f *fakeStore) MarkSent(_ context.Context, id uint, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	reminder, ok := f.reminders[id]
	if !ok || reminder.SentAt != nil {
		return repository.ErrAlreadySent
	}
	sent := at
	reminder.SentAt = &sent
	return nil
}

func (f *fakeStore) UpdateReminder(_ context.Context, reminder *domain.Reminder) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.reminders[reminder.ID]; !ok {
		return repository.ErrNotFound
	}
	reminder.UpdatedAt = time.Now().UTC()
	f.reminders[reminder.ID] = reminder
	return nil
}

func (f *fakeStore) DeleteReminder(_ context.Context, id uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.reminders[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.reminders, id)
	return nil
}

// --- helpers internos do fake ---

func (f *fakeStore) deleteColumnCards(columnID uint) {
	for cid, card := range f.cards {
		if card.ColumnID == columnID {
			for rid, reminder := range f.reminders {
				if reminder.CardID == cid {
					delete(f.reminders, rid)
				}
			}
			delete(f.cards, cid)
		}
	}
}

// columnsOf monta as colunas de um board com seus cards (e reminders), ordenados
// por position, espelhando o Preload + ORDER BY do Store.
func (f *fakeStore) columnsOf(boardID uint) []domain.Column {
	var columns []domain.Column
	for _, column := range f.columns {
		if column.BoardID == boardID {
			columns = append(columns, *column)
		}
	}
	sortColumns(columns)
	for i := range columns {
		columns[i].Cards = f.cardsOf(columns[i].ID)
	}
	return columns
}

func (f *fakeStore) cardsOf(columnID uint) []domain.Card {
	var cards []domain.Card
	for _, card := range f.cards {
		if card.ColumnID == columnID {
			cards = append(cards, *card)
		}
	}
	sortCards(cards)
	for i := range cards {
		cards[i].Reminders = f.remindersOf(cards[i].ID)
	}
	return cards
}

func (f *fakeStore) remindersOf(cardID uint) []domain.Reminder {
	var reminders []domain.Reminder
	for _, reminder := range f.reminders {
		if reminder.CardID == cardID {
			reminders = append(reminders, *reminder)
		}
	}
	return reminders
}

// sortColumns/sortCards espelham o ORDER BY position ASC, id ASC do Store.
func sortColumns(columns []domain.Column) {
	slices.SortFunc(columns, func(a, b domain.Column) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}
		return int(a.ID) - int(b.ID)
	})
}

func sortCards(cards []domain.Card) {
	slices.SortFunc(cards, func(a, b domain.Card) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}
		return int(a.ID) - int(b.ID)
	})
}
