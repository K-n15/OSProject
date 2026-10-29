package api

import (
	"net/http"

	"itassist/internal/store"
)

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request, u store.User) {
	res, err := s.Store.ListNotifications(r.Context(), u.ID, 20)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) markNotificationsRead(w http.ResponseWriter, r *http.Request, u store.User) {
	var in struct {
		TicketID int64 `json:"ticketId"`
	}
	if r.ContentLength != 0 && !decode(w, r, &in) {
		return
	}
	if in.TicketID < 0 {
		writeErr(w, http.StatusBadRequest, "invalid ticketId")
		return
	}
	if err := s.Store.MarkNotificationsRead(r.Context(), u.ID, in.TicketID); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
