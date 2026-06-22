package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/Arthur-Queiroz/jboard/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server amarra as dependências dos handlers. Os campos são as interfaces de
// repositório, então em teste pode-se injetar mocks.
type Server struct {
	Boards           repository.BoardRepository
	Columns          repository.ColumnRepository
	Cards            repository.CardRepository
	Reminders        repository.ReminderRepository
	DefaultRecipient string
	APIToken         string
	// AllowedOrigins é a allowlist de CORS (ver corsMiddleware). Vazia = nenhum
	// header de CORS é emitido (comportamento same-origin padrão).
	AllowedOrigins []string
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(s.corsMiddleware)
	r.Use(s.authMiddleware)

	r.Get("/api/health", s.health)

	r.Route("/api", func(r chi.Router) {
		r.Route("/boards", func(r chi.Router) {
			r.Get("/", s.listBoards)
			r.Post("/", s.createBoard)
			r.Get("/{boardID}", s.getBoard)
			r.Put("/{boardID}", s.updateBoard)
			r.Delete("/{boardID}", s.deleteBoard)
			r.Get("/{boardID}/columns", s.listColumns)
			r.Post("/{boardID}/columns", s.createColumn)
		})
		r.Route("/columns", func(r chi.Router) {
			r.Get("/{columnID}", s.getColumn)
			r.Put("/{columnID}", s.updateColumn)
			r.Delete("/{columnID}", s.deleteColumn)
			r.Get("/{columnID}/cards", s.listCards)
			r.Post("/{columnID}/cards", s.createCard)
			// Reorder DnD: fixa a ordem dos cards da coluna após um drop.
			r.Post("/{columnID}/cards/reorder", s.reorderCards)
		})
		r.Route("/cards", func(r chi.Router) {
			r.Get("/{cardID}", s.getCard)
			r.Put("/{cardID}", s.updateCard)
			r.Delete("/{cardID}", s.deleteCard)
			r.Get("/{cardID}/reminders", s.listReminders)
			r.Post("/{cardID}/reminders", s.createReminder)
		})
		r.Route("/reminders", func(r chi.Router) {
			r.Get("/{reminderID}", s.getReminder)
			r.Put("/{reminderID}", s.updateReminder)
			r.Delete("/{reminderID}", s.deleteReminder)
		})
	})

	return r
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- helpers ---

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func respondError(w http.ResponseWriter, status int, err error) {
	respondJSON(w, status, map[string]string{"error": err.Error()})
}

// respondRepoError mapeia ErrNotFound do repositório para 404; resto vira 500.
func respondRepoError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		respondError(w, http.StatusNotFound, err)
		return
	}
	respondError(w, http.StatusInternalServerError, err)
}

func parseID(r *http.Request, key string) (uint, error) {
	n, err := strconv.ParseUint(chi.URLParam(r, key), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}
