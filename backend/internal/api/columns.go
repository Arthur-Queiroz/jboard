package api

import (
	"net/http"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
)

func (s *Server) createColumn(w http.ResponseWriter, r *http.Request) {
	boardID, err := parseID(r, "boardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var column domain.Column
	if err := decodeJSON(r, &column); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("title", column.Title); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	column.ID = 0
	column.BoardID = boardID
	if err := s.Columns.CreateColumn(r.Context(), &column); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, column)
}

func (s *Server) listColumns(w http.ResponseWriter, r *http.Request) {
	boardID, err := parseID(r, "boardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	columns, err := s.Columns.ListColumns(r.Context(), boardID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, columns)
}

func (s *Server) getColumn(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "columnID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	column, err := s.Columns.GetColumn(r.Context(), id)
	if err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, column)
}

func (s *Server) updateColumn(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "columnID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var column domain.Column
	if err := decodeJSON(r, &column); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("title", column.Title); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	column.ID = id
	if err := s.Columns.UpdateColumn(r.Context(), &column); err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, column)
}

func (s *Server) deleteColumn(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "columnID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.Columns.DeleteColumn(r.Context(), id); err != nil {
		respondRepoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
