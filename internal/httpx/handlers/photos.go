package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/RENCHILIU/gallerio/internal/httpx/middleware"
	"github.com/RENCHILIU/gallerio/internal/store"
)

type PhotosHandler struct {
	Store           *store.Store
	DefaultPageSize int // e.g. 50
	MaxPageSize     int // e.g. 500
}

func NewPhotosHandler(s *store.Store, defaultPageSize, maxPageSize int) *PhotosHandler {
	return &PhotosHandler{Store: s, DefaultPageSize: defaultPageSize, MaxPageSize: maxPageSize}
}

type errorResp struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// GET /api/photos
func (h *PhotosHandler) List(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.RequestIDFromContext(r.Context())

	limit := h.DefaultPageSize
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > h.MaxPageSize {
			writeJSON(w, http.StatusBadRequest, errorResp{
				Code: "BAD_LIMIT", Message: "limit must be 1.." + strconv.Itoa(h.MaxPageSize), RequestID: reqID,
			})
			return
		}
		limit = n
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusBadRequest, errorResp{
				Code: "BAD_OFFSET", Message: "offset must be >= 0", RequestID: reqID,
			})
			return
		}
		offset = n
	}

	res, err := h.Store.ListPhotos(r.Context(), limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResp{
			Code: "INTERNAL", Message: "database error", RequestID: reqID,
		})
		return
	}

	type photoDTO struct {
		ID           int64   `json:"id"`
		FileName     string  `json:"fileName"`
		MimeType     *string `json:"mimeType,omitempty"`
		SizeBytes    int64   `json:"sizeBytes"`
		PathOriginal string  `json:"pathOriginal"`
		UploadedAt   string  `json:"uploadedAt"` // RFC3339
	}

	out := struct {
		Items   []photoDTO `json:"items"`
		Count   int        `json:"count"`
		Limit   int        `json:"limit"`
		Offset  int        `json:"offset"`
		Total   int        `json:"total"`
		HasMore bool       `json:"hasMore"`
	}{
		Items:   make([]photoDTO, 0, len(res.Items)),
		Count:   len(res.Items),
		Limit:   limit,
		Offset:  offset,
		Total:   res.Total,
		HasMore: offset+len(res.Items) < res.Total,
	}

	for _, p := range res.Items {
		out.Items = append(out.Items, photoDTO{
			ID:           p.ID,
			FileName:     p.FileName,
			MimeType:     p.MimeType,
			SizeBytes:    p.SizeBytes,
			PathOriginal: p.PathOriginal,
			UploadedAt:   p.UploadedAt.UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
