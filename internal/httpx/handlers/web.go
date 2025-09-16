package handlers

import (
	"html/template"
	"net/http"
	"strconv"
)

type WebHandler struct {
	T           *template.Template
	PageSize    int
	UploadMaxMB int
}

func NewWebHandler(t *template.Template, pageSize, uploadMaxMB int) *WebHandler {
	return &WebHandler{T: t, PageSize: pageSize, UploadMaxMB: uploadMaxMB}
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.T.ExecuteTemplate(w, "index", map[string]any{
		"PageSize":    h.PageSize,
		"UploadMaxMB": h.UploadMaxMB,
	})
}

// 新增：/slideshow 随机播放页
func (h *WebHandler) Slideshow(w http.ResponseWriter, r *http.Request) {
	// 读取 ?interval= 秒（1..60），默认 3s。也可以在前端再调整。
	interval := 3
	if v := r.URL.Query().Get("interval"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 60 {
			interval = n
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.T.ExecuteTemplate(w, "slideshow", map[string]any{
		"PageSize":    h.PageSize, // 用于前端分页抓全量
		"IntervalSec": interval,   // 初始播放间隔
	})
}
