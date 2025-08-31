package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Recover panic -> 500
func Recover(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "err", rec)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Logging 基础访问日志
func Logging(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &wrapResp{ResponseWriter: w, code: 200}
		next.ServeHTTP(ww, r)
		slog.Info("http",
			"method", r.Method, "path", r.URL.Path,
			"status", ww.code, "dur", time.Since(start).String(),
			"remote", r.RemoteAddr)
	}
	return http.HandlerFunc(fn)
}

type wrapResp struct {
	http.ResponseWriter
	code int
}
func (w *wrapResp) WriteHeader(code int) { w.code = code; w.ResponseWriter.WriteHeader(code) }

// NoDirFS：禁止目录列出（用于 /media）
type noDirFS struct{ http.Dir }
func (fs noDirFS) Open(name string) (http.File, error) {
	f, err := fs.Dir.Open(name)
	if err != nil { return nil, err }
	stat, err := f.Stat()
	if err != nil { return nil, err }
	if stat.IsDir() { // 禁止目录浏览
		return nil, os.ErrNotExist
	}
	return f, nil
}
func NoDirFileServer(dir string) http.Handler {
	return http.FileServer(noDirFS{http.Dir(dir)})
}

