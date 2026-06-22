package api

import (
	"net/http"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
)

func (s *Server) createCard(w http.ResponseWriter, r *http.Request) {
	columnID, err := parseID(r, "columnID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var card domain.Card
	if err := decodeJSON(r, &card); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("title", card.Title); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	card.ID = 0
	card.ColumnID = columnID
	if err := s.Cards.CreateCard(r.Context(), &card); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	// Garante reminders: [] no JSON (nil slice vira "null" no encoding/json,
	// o que quebra o frontend que faz card.reminders.length). Card recém-criado
	// não tem lembretes, então [] é fiel à realidade.
	card.Reminders = []domain.Reminder{}
	respondJSON(w, http.StatusCreated, card)
}

func (s *Server) listCards(w http.ResponseWriter, r *http.Request) {
	columnID, err := parseID(r, "columnID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	cards, err := s.Cards.ListCards(r.Context(), columnID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, cards)
}

func (s *Server) getCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "cardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	card, err := s.Cards.GetCard(r.Context(), id)
	if err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, card)
}

func (s *Server) updateCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "cardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var card domain.Card
	if err := decodeJSON(r, &card); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("title", card.Title); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	card.ID = id
	if err := s.Cards.UpdateCard(r.Context(), &card); err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, card)
}

func (s *Server) deleteCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "cardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.Cards.DeleteCard(r.Context(), id); err != nil {
		respondRepoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// reorderCards fixa a ordem dos cards da coluna após um drag-and-drop. Corpo:
// {"card_ids": [3, 1, 2]} — o card na posição 0 do slice vira position=0, etc.
// Para um move entre colunas, o frontend chama reorder na origem (sem o card
// movido) e no destino (com o card movido na nova posição).
func (s *Server) reorderCards(w http.ResponseWriter, r *http.Request) {
	columnID, err := parseID(r, "columnID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var body struct {
		CardIDs []uint `json:"card_ids"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.Cards.ReorderCards(r.Context(), columnID, body.CardIDs); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
