package api

import (
	"net/http"

	"github.com/Arthur-Queiroz/jboard/internal/domain"
)

func (s *Server) createReminder(w http.ResponseWriter, r *http.Request) {
	cardID, err := parseID(r, "cardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var reminder domain.Reminder
	if err := decodeJSON(r, &reminder); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := futureTime("reminder_at", reminder.ReminderAt); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("message", reminder.Message); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	reminder.ID = 0
	reminder.CardID = cardID
	reminder.SentAt = nil
	// Se o card não trouxer destinatário, usa o destinatário padrão da config.
	if reminder.Recipient == "" {
		reminder.Recipient = s.DefaultRecipient
	}
	if err := s.Reminders.CreateReminder(r.Context(), &reminder); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusCreated, reminder)
}

func (s *Server) listReminders(w http.ResponseWriter, r *http.Request) {
	cardID, err := parseID(r, "cardID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	reminders, err := s.Reminders.ListReminders(r.Context(), cardID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, reminders)
}

func (s *Server) getReminder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "reminderID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	reminder, err := s.Reminders.GetReminder(r.Context(), id)
	if err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, reminder)
}

func (s *Server) updateReminder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "reminderID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	var reminder domain.Reminder
	if err := decodeJSON(r, &reminder); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := futureTime("reminder_at", reminder.ReminderAt); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := required("message", reminder.Message); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	reminder.ID = id
	if err := s.Reminders.UpdateReminder(r.Context(), &reminder); err != nil {
		respondRepoError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, reminder)
}

func (s *Server) deleteReminder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "reminderID")
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.Reminders.DeleteReminder(r.Context(), id); err != nil {
		respondRepoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
