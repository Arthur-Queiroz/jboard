package api

import (
	"net/http"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
)

func (s *Server) createBoard(w http.ResponseWriter, r *http.Request) {
	var board domain.Board
	if err := decodeJSON(r, &board); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("title", board.Title); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	board.ID = 0
	if err := s.Boards.CreateBoard(r.Context(), &board); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, board)
}

func (s *Server) listBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := s.Boards.ListBoards(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, boards)
}

func (s *Server) getBoard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "boardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	board, err := s.Boards.GetBoard(r.Context(), id)
	if err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, board)
}

func (s *Server) updateBoard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "boardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var board domain.Board
	if err := decodeJSON(r, &board); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("title", board.Title); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	board.ID = id
	if err := s.Boards.UpdateBoard(r.Context(), &board); err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, board)
}

func (s *Server) deleteBoard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "boardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.Boards.DeleteBoard(r.Context(), id); err != nil {
		respondRepoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
