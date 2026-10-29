package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"crypto/rand"
	"encoding/hex"

	"itassist/internal/store"
)

const (
	maxImageBytes = 2 << 20
)

var imageExt = map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/gif": "gif", "image/webp": "webp"}
var validStatus = map[string]bool{"pending": true, "ongoing": true, "closed": true}

func (s *Server) me(w http.ResponseWriter, r *http.Request, u store.User) { writeJSON(w, 200, u) }

func (s *Server) devices(w http.ResponseWriter, r *http.Request, _ store.User) {
	v, err := s.Store.Devices(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) resolveTypes(w http.ResponseWriter, r *http.Request, _ store.User) {
	v, err := s.Store.ResolveTypes(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) listTickets(w http.ResponseWriter, r *http.Request, u store.User) {
	q := r.URL.Query()
	status := q.Get("status")
	if status != "" && !validStatus[status] {
		writeErr(w, http.StatusBadRequest, "status must be pending, ongoing or closed")
		return
	}
	page, size := pageParams(q.Get("page"), q.Get("pageSize"))
	res, err := s.Store.ListTickets(r.Context(), store.ListParams{
		User: u, Status: status, Query: strings.TrimSpace(q.Get("q")), Page: page, PageSize: size,
	})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) getTicket(w http.ResponseWriter, r *http.Request, u store.User) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	t, err := s.Store.GetTicket(r.Context(), u, id)
	if err != nil {
		fail(w, err)
		return
	}
	for i := range t.Attachments {
		t.Attachments[i].URL = fmt.Sprintf("/api/tickets/%d/attachments/%d", t.ID, t.Attachments[i].ID)
	}
	writeJSON(w, 200, t)
}

func (s *Server) getAttachment(w http.ResponseWriter, r *http.Request, u store.User) {
	ticketID, ok := idParam(w, r)
	if !ok {
		return
	}
	attID, err := strconv.ParseInt(r.PathValue("aid"), 10, 64)
	if err != nil || attID < 1 {
		writeErr(w, http.StatusBadRequest, "invalid attachment id")
		return
	}
	a, err := s.Store.GetAttachment(r.Context(), u, ticketID, attID)
	if err != nil {
		fail(w, err)
		return
	}
	body, err := s.Files.Get(r.Context(), a.ObjectKey)
	if err != nil {
		fail(w, err)
		return
	}
	defer body.Close()

	h := w.Header()
	h.Set("Content-Type", a.MimeType)
	h.Set("Content-Length", strconv.Itoa(a.SizeBytes))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Disposition", "inline")
	h.Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil {
		log.Printf("attachment %d: %v", attID, err)
	}
}

func (s *Server) createTicket(w http.ResponseWriter, r *http.Request, u store.User) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeErr(w, http.StatusRequestEntityTooLarge, "request too large or malformed (images must be under 2 MB)")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	desc := strings.TrimSpace(r.FormValue("description"))
	deviceID, err := strconv.Atoi(r.FormValue("device_id"))
	if title == "" || len(title) > 80 || desc == "" || len(desc) > 5000 || err != nil {
		writeErr(w, http.StatusBadRequest, "title (max 80), device_id and description are required")
		return
	}

	var img []byte
	var mime, ext string
	file, _, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		img, err = io.ReadAll(io.LimitReader(file, maxImageBytes+1))
		if err != nil || len(img) == 0 || len(img) > maxImageBytes {
			writeErr(w, http.StatusRequestEntityTooLarge, "image must be between 1 byte and 2 MB")
			return
		}
		head := img
		if len(head) > 512 {
			head = head[:512]
		}
		mime = http.DetectContentType(head)
		var ok bool
		if ext, ok = imageExt[mime]; !ok {
			writeErr(w, http.StatusUnsupportedMediaType, "image must be PNG, JPEG, GIF or WebP")
			return
		}
	} else if err != http.ErrMissingFile {
		writeErr(w, http.StatusBadRequest, "could not read the image")
		return
	}

	id, err := s.Store.CreateTicket(r.Context(), u.ID, title, deviceID, desc)
	if err != nil {
		fail(w, err)
		return
	}
	if img != nil {
		key := fmt.Sprintf("tickets/%d/%s.%s", id, randomName(), ext)
		if err := s.Files.Put(r.Context(), key, mime, img); err != nil {
			_ = s.Store.DeleteTicket(r.Context(), id)
			fail(w, err)
			return
		}
		if err := s.Store.AddAttachment(r.Context(), id, key, mime, len(img)); err != nil {
			_ = s.Files.Delete(r.Context(), key)
			_ = s.Store.DeleteTicket(r.Context(), id)
			fail(w, err)
			return
		}
	}
	t, err := s.Store.GetTicket(r.Context(), u, id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func randomName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) addMessage(w http.ResponseWriter, r *http.Request, u store.User) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" || len(in.Body) > 5000 {
		writeErr(w, http.StatusBadRequest, "body is required (max 5000 characters)")
		return
	}
	if err := s.Store.AddMessage(r.Context(), u, id, in.Body); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) acceptTicket(w http.ResponseWriter, r *http.Request, u store.User) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := s.Store.Accept(r.Context(), u, id); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) closeTicket(w http.ResponseWriter, r *http.Request, u store.User) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in struct {
		ResolveTypeID int    `json:"resolveTypeId"`
		Note          string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.ResolveTypeID < 1 || len(in.Note) > 5000 {
		writeErr(w, http.StatusBadRequest, "resolveTypeId is required; note max 5000 characters")
		return
	}
	if err := s.Store.Close(r.Context(), u, id, in.ResolveTypeID, strings.TrimSpace(in.Note)); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pageParams(pageStr, sizeStr string) (int, int) {
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(sizeStr)
	if size < 1 {
		size = 10
	}
	if size > 50 {
		size = 50
	}
	return page, size
}
